package handler

// 评论防护（黑名单/批量，契约 #69-72）与通知中心（契约 #73-75）。

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------- 黑名单（#69-71） ----------

// AdminListBlacklist GET /api/v1/admin/comment-blacklist?type=&page=&pageSize=
func AdminListBlacklist(c *gin.Context) {
	pq := common.ParsePage(c, 20)
	typeValue := c.Query("type")
	if typeValue != "" && typeValue != "ip" && typeValue != "email" && typeValue != "keyword" {
		common.Fail(c, common.CodeParamError, "type 仅支持 ip / email / keyword")
		return
	}

	buildQuery := func() *gorm.DB {
		db := model.DB.Model(&model.CommentBlacklist{})
		if typeValue != "" {
			db = db.Where("type = ?", typeValue)
		}
		return db
	}
	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var list []model.CommentBlacklist
	if err := buildQuery().Order("created_at DESC, id DESC").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&list).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, pq.Data(list, total))
}

// AdminCreateBlacklist POST /api/v1/admin/comment-blacklist
func AdminCreateBlacklist(c *gin.Context) {
	var req struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	req.Type = strings.TrimSpace(req.Type)
	req.Value = strings.TrimSpace(req.Value)
	switch {
	case req.Type != "ip" && req.Type != "email" && req.Type != "keyword":
		common.Fail(c, common.CodeParamError, "type 仅支持 ip / email / keyword")
		return
	case req.Value == "" || utf8.RuneCountInString(req.Value) > 200:
		common.Fail(c, common.CodeParamError, "value 必填且不超过 200 字")
		return
	}
	var n int64
	model.DB.Model(&model.CommentBlacklist{}).
		Where("type = ? AND value = ?", req.Type, req.Value).Count(&n)
	if n > 0 {
		common.Fail(c, common.CodeParamError, "该条目已在黑名单中")
		return
	}
	item := model.CommentBlacklist{Type: req.Type, Value: req.Value}
	if err := model.DB.Create(&item).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"item": item})
}

// AdminDeleteBlacklist DELETE /api/v1/admin/comment-blacklist/:id
func AdminDeleteBlacklist(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var item model.CommentBlacklist
	if err := model.DB.First(&item, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "黑名单条目不存在")
		return
	}
	if err := model.DB.Delete(&item).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// ---------- 评论批量操作（#72） ----------

// AdminBatchComments POST /api/v1/admin/comments/batch
// action: approve→1 / reject→2 / spam→3 / delete→物理删除（含 children）
func AdminBatchComments(c *gin.Context) {
	var req struct {
		Action string `json:"action"`
		IDs    []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > 100 {
		common.Fail(c, common.CodeParamError, "ids 数量需为 1~100")
		return
	}
	switch req.Action {
	case "approve":
		batchUpdateCommentStatus(req.IDs, model.CommentApproved)
		writeAudit(c, "comment.batch", "comment", "", fmt.Sprintf("approve %d 条", len(req.IDs)))
	case "reject":
		batchUpdateCommentStatus(req.IDs, model.CommentRejected)
		writeAudit(c, "comment.batch", "comment", "", fmt.Sprintf("reject %d 条", len(req.IDs)))
	case "spam":
		batchUpdateCommentStatus(req.IDs, model.CommentSpam)
		writeAudit(c, "comment.batch", "comment", "", fmt.Sprintf("spam %d 条", len(req.IDs)))
	case "delete":
		ids, err := collectCommentSubtree(req.IDs)
		if err != nil {
			common.ServerError(c, err)
			return
		}
		if err := model.DB.Where("id IN ?", ids).Delete(&model.Comment{}).Error; err != nil {
			common.ServerError(c, err)
			return
		}
		common.OK(c, gin.H{"updated": int64(len(req.IDs))})
		return
	default:
		common.Fail(c, common.CodeParamError, "action 仅支持 approve/reject/spam/delete")
		return
	}
	common.OK(c, gin.H{"updated": int64(len(req.IDs))})
}

func batchUpdateCommentStatus(ids []uint, status int8) {
	model.DB.Model(&model.Comment{}).Where("id IN ?", ids).Update("status", status)
}

// collectCommentSubtree 批量删除时把所有子孙一并收集（复用单条删除的级联思路）
func collectCommentSubtree(ids []uint) ([]uint, error) {
	all := append([]uint{}, ids...)
	frontier := append([]uint{}, ids...)
	for len(frontier) > 0 {
		var children []uint
		if err := model.DB.Model(&model.Comment{}).
			Where("parent_id IN ?", frontier).Pluck("id", &children).Error; err != nil {
			return nil, err
		}
		if len(children) == 0 {
			break
		}
		all = append(all, children...)
		frontier = children
	}
	return all, nil
}

// ---------- 通知中心（#73-75） ----------

// AdminListNotifications GET /api/v1/admin/notifications —— 最新 10 条 + 未读数
func AdminListNotifications(c *gin.Context) {
	var list []model.Notification
	if err := model.DB.Order("id DESC").Limit(10).Find(&list).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var unread int64
	// 反引号包住 read：MySQL 8 保留字，裸写会触发 Error 1064
	if err := model.DB.Model(&model.Notification{}).Where("`read` = ?", false).Count(&unread).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"list": list, "unreadCount": unread})
}

// AdminMarkAllNotificationsRead PUT /api/v1/admin/notifications/read-all
func AdminMarkAllNotificationsRead(c *gin.Context) {
	if err := model.DB.Model(&model.Notification{}).Where("`read` = ?", false).
		Update("read", true).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// AdminMarkNotificationRead PUT /api/v1/admin/notifications/:id/read
func AdminMarkNotificationRead(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := model.DB.Model(&model.Notification{}).Where("id = ?", id).
		Update("read", true).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}
