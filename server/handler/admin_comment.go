package handler

import (
	"strings"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminListComments GET /api/v1/admin/comments —— status/postId 过滤，最新在前
func AdminListComments(c *gin.Context) {
	pq := common.ParsePage(c, 10)

	statusParam := c.Query("status")
	statusVal := int8(-1)
	if statusParam != "" {
		if len(statusParam) != 1 || statusParam[0] < '0' || statusParam[0] > '4' {
			common.Fail(c, common.CodeParamError, "status 参数不合法")
			return
		}
		statusVal = int8(statusParam[0] - '0')
	}
	postID := queryUint(c, "postId")

	buildQuery := func() *gorm.DB {
		db := model.DB.Model(&model.Comment{})
		if statusParam != "" {
			db = db.Where("status = ?", statusVal)
		}
		if postID > 0 {
			db = db.Where("post_id = ?", postID)
		}
		return db
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var comments []model.Comment
	if err := buildQuery().Order("id DESC").Offset(pq.Offset()).Limit(pq.PageSize).
		Find(&comments).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	postIDs := make([]uint, 0, len(comments))
	for i := range comments {
		postIDs = append(postIDs, comments[i].PostID)
	}
	titles := postTitleMap(postIDs)

	list := make([]model.CommentAdminDTO, 0, len(comments))
	for i := range comments {
		list = append(list, model.ToCommentAdmin(&comments[i], titles[comments[i].PostID]))
	}
	common.OK(c, pq.Data(list, total))
}

// AdminUpdateCommentStatus PUT /api/v1/admin/comments/:id/status —— status 允许 0~4
// （0=恢复待审，3=标记垃圾，4=移入回收站；删除走 DELETE）
func AdminUpdateCommentStatus(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req struct {
		Status int8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if req.Status < model.CommentPending || req.Status > model.CommentTrash {
		common.Fail(c, common.CodeParamError, "status 取值不合法")
		return
	}

	var comment model.Comment
	if err := model.DB.First(&comment, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "评论不存在")
		return
	}
	if err := model.DB.Model(&comment).Update("status", req.Status).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// AdminReplyComment POST /api/v1/admin/comments/:id/reply —— 管理员回复，isAdmin=1 直接通过
func AdminReplyComment(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		common.Fail(c, common.CodeParamError, "回复内容不能为空")
		return
	}
	if utf8.RuneCountInString(req.Content) > 1000 {
		common.Fail(c, common.CodeParamError, "回复内容不能超过 1000 字")
		return
	}

	user, okUser := currentUser(c)
	if !okUser {
		common.Unauthorized(c)
		return
	}

	var target model.Comment
	if err := model.DB.First(&target, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "评论不存在")
		return
	}

	reply := model.Comment{
		PostID:   target.PostID,
		ParentID: target.ID,
		Nickname: user.Nickname,
		Email:    user.Email,
		Content:  req.Content,
		Status:   model.CommentApproved, // 管理员回复直接通过
		IP:       c.ClientIP(),
		IsAdmin:  true,
	}
	if err := model.DB.Create(&reply).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	titles := postTitleMap([]uint{reply.PostID})
	common.OK(c, gin.H{"comment": model.ToCommentAdmin(&reply, titles[reply.PostID])})
}

// AdminDeleteComment DELETE /api/v1/admin/comments/:id —— 连同其全部子孙一起删除
func AdminDeleteComment(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var comment model.Comment
	if err := model.DB.First(&comment, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "评论不存在")
		return
	}

	// 逐层收集子孙（数据结构上允许深层回复，防御式级联）
	ids := []uint{comment.ID}
	frontier := []uint{comment.ID}
	for len(frontier) > 0 {
		var children []uint
		if err := model.DB.Model(&model.Comment{}).
			Where("parent_id IN ?", frontier).Pluck("id", &children).Error; err != nil {
			common.ServerError(c, err)
			return
		}
		if len(children) == 0 {
			break
		}
		ids = append(ids, children...)
		frontier = children
	}

	if err := model.DB.Where("id IN ?", ids).Delete(&model.Comment{}).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}
