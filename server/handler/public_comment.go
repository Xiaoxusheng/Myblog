package handler

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/middleware"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListComments GET /api/v1/posts/:slug/comments —— 仅已通过，两级树（顶级倒序、children 升序）
func ListComments(c *gin.Context) {
	post, found, err := resolvePublishedPost(c.Param("slug"))
	if err != nil {
		common.ServerError(c, err)
		return
	}
	if !found {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}

	var comments []model.Comment
	if err := model.DB.Where("post_id = ? AND status = ?", post.ID, model.CommentApproved).
		Order("id ASC").Find(&comments).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"list": buildCommentTree(comments)})
}

// buildCommentTree 构建两级评论树：
// parent 直接缺失或不可见的通过评论提升为顶级；
// 深层回复统一扁平挂到顶级祖先（保证契约的两级结构）。
func buildCommentTree(comments []model.Comment) []model.CommentPublicDTO {
	nodes := make(map[uint]*model.CommentPublicDTO, len(comments))
	byID := make(map[uint]*model.Comment, len(comments))
	for i := range comments {
		cm := &comments[i]
		byID[cm.ID] = cm
		nodes[cm.ID] = &model.CommentPublicDTO{
			ID:        cm.ID,
			ParentID:  cm.ParentID,
			Nickname:  cm.Nickname,
			Website:   cm.Website,
			Content:   cm.Content,
			IsAdmin:   cm.IsAdmin,
			CreatedAt: cm.CreatedAt,
			Children:  []model.CommentPublicDTO{},
		}
	}

	tops := make([]*model.CommentPublicDTO, 0)
	for i := range comments {
		cm := &comments[i]
		if cm.ParentID == 0 {
			tops = append(tops, nodes[cm.ID])
			continue
		}
		rootID, ok := commentRootID(cm.ParentID, byID)
		if !ok {
			tops = append(tops, nodes[cm.ID])
			continue
		}
		root := nodes[rootID]
		root.Children = append(root.Children, *nodes[cm.ID])
	}

	// 顶级倒序（新在前），children 升序（对话顺序）
	sort.SliceStable(tops, func(i, j int) bool {
		if !tops[i].CreatedAt.Equal(tops[j].CreatedAt) {
			return tops[i].CreatedAt.After(tops[j].CreatedAt)
		}
		return tops[i].ID > tops[j].ID
	})
	for _, t := range tops {
		children := t.Children
		sort.SliceStable(children, func(i, j int) bool {
			if !children[i].CreatedAt.Equal(children[j].CreatedAt) {
				return children[i].CreatedAt.Before(children[j].CreatedAt)
			}
			return children[i].ID < children[j].ID
		})
	}

	list := make([]model.CommentPublicDTO, 0, len(tops))
	for _, t := range tops {
		list = append(list, *t)
	}
	return list
}

// commentRootID 沿 parent 链向上找顶级评论 id；链断裂或过深返回 false
func commentRootID(parentID uint, byID map[uint]*model.Comment) (uint, bool) {
	cur := parentID
	for depth := 0; depth < 64; depth++ {
		parent, ok := byID[cur]
		if !ok {
			return 0, false
		}
		if parent.ParentID == 0 {
			return parent.ID, true
		}
		cur = parent.ParentID
	}
	return 0, false
}

// CreateComment POST /api/v1/posts/:slug/comments —— 游客评论：待审核，限流 20003，关闭 20002
func CreateComment(c *gin.Context) {
	post, found, err := resolvePublishedPost(c.Param("slug"))
	if err != nil {
		common.ServerError(c, err)
		return
	}
	if !found {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}

	if !getSettingsDTO(model.DB).CommentEnabled {
		common.Fail(c, common.CodeCommentClosed, "评论已关闭")
		return
	}

	var req struct {
		ParentID uint   `json:"parentId"`
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Website  string `json:"website"`
		Content  string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Email = strings.TrimSpace(req.Email)
	req.Website = strings.TrimSpace(req.Website)
	req.Content = strings.TrimSpace(req.Content)

	switch {
	case req.Nickname == "" || req.Email == "" || req.Content == "":
		common.Fail(c, common.CodeParamError, "昵称、邮箱、内容为必填项")
		return
	case !strings.Contains(req.Email, "@") || utf8.RuneCountInString(req.Email) > 128:
		common.Fail(c, common.CodeParamError, "邮箱格式不正确")
		return
	case utf8.RuneCountInString(req.Nickname) > 64:
		common.Fail(c, common.CodeParamError, "昵称不能超过 64 字")
		return
	case utf8.RuneCountInString(req.Website) > 256:
		common.Fail(c, common.CodeParamError, "网站地址过长")
		return
	case utf8.RuneCountInString(req.Content) > 1000:
		common.Fail(c, common.CodeParamError, "评论内容不能超过 1000 字")
		return
	}

	if req.ParentID > 0 {
		var parent model.Comment
		err := model.DB.Where("id = ? AND post_id = ?", req.ParentID, post.ID).First(&parent).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.Fail(c, common.CodeParamError, "回复的评论不存在")
			return
		}
		if err != nil {
			common.ServerError(c, err)
			return
		}
	}

	// 垃圾检测：黑名单/重复/链接过多/高频 → status=3（契约 #5）
	status := model.CommentPending
	if isSpam(model.DB, post.ID, req.Nickname, req.Email, req.Content, c.ClientIP()) {
		status = model.CommentSpam
	}

	cm := model.Comment{
		PostID:   post.ID,
		ParentID: req.ParentID,
		Nickname: req.Nickname,
		Email:    req.Email,
		Website:  req.Website,
		Content:  req.Content,
		Status:   status,
		IP:       c.ClientIP(),
	}
	if err := model.DB.Create(&cm).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	// 仅成功入库才占用限流窗口
	middleware.MarkComment(c)

	// 通知中心（契约 #73）：垃圾评论与待审评论分别提醒
	notifyCommentCreated(model.DB, post.Title, req.Content, status == model.CommentSpam)

	common.OK(c, gin.H{"comment": gin.H{
		"id":        cm.ID,
		"parentId":  cm.ParentID,
		"nickname":  cm.Nickname,
		"website":   cm.Website,
		"content":   cm.Content,
		"createdAt": cm.CreatedAt,
	}})
}
