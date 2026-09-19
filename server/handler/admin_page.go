package handler

import (
	"strings"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// ---------- 自定义页面（契约 #38-42） ----------

// AdminListPages GET /api/v1/admin/pages
func AdminListPages(c *gin.Context) {
	pq := common.ParsePage(c, 10)

	var total int64
	if err := model.DB.Model(&model.Page{}).Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var pages []model.Page
	if err := model.DB.Order("id DESC").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&pages).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, pq.Data(pages, total))
}

// pagePayload 创建/更新页面入参（slug 必填唯一）
type pagePayload struct {
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
	Status  int8   `json:"status"`
}

func (p *pagePayload) validate(c *gin.Context) bool {
	switch {
	case strings.TrimSpace(p.Title) == "":
		common.Fail(c, common.CodeParamError, "标题不能为空")
	case strings.TrimSpace(p.Slug) == "":
		common.Fail(c, common.CodeParamError, "slug 不能为空")
	case utf8.RuneCountInString(strings.TrimSpace(p.Title)) > 200:
		common.Fail(c, common.CodeParamError, "标题不能超过 200 字")
	case utf8.RuneCountInString(strings.TrimSpace(p.Slug)) > 200:
		common.Fail(c, common.CodeParamError, "slug 不能超过 200 字符")
	case p.Status != 0 && p.Status != 1:
		common.Fail(c, common.CodeParamError, "status 仅允许 0（草稿）或 1（已发布）")
	default:
		return true
	}
	return false
}

func pageSlugTaken(slug string, excludeID uint) bool {
	var n int64
	model.DB.Model(&model.Page{}).Where("slug = ? AND id <> ?", slug, excludeID).Count(&n)
	return n > 0
}

// AdminCreatePage POST /api/v1/admin/pages
func AdminCreatePage(c *gin.Context) {
	var req pagePayload
	if err := c.ShouldBindJSON(&req); err != nil || !req.validate(c) {
		if err != nil {
			common.Fail(c, common.CodeParamError, "参数错误")
		}
		return
	}
	slug := strings.TrimSpace(req.Slug)
	if pageSlugTaken(slug, 0) {
		common.Fail(c, common.CodeParamError, "slug 已存在")
		return
	}

	page := model.Page{
		Title:   strings.TrimSpace(req.Title),
		Slug:    slug,
		Content: req.Content,
		Status:  req.Status,
	}
	if err := model.DB.Create(&page).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, page)
}

// AdminGetPage GET /api/v1/admin/pages/:id
func AdminGetPage(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}
	common.OK(c, gin.H{"page": page})
}

// AdminUpdatePage PUT /api/v1/admin/pages/:id
func AdminUpdatePage(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req pagePayload
	if err := c.ShouldBindJSON(&req); err != nil || !req.validate(c) {
		if err != nil {
			common.Fail(c, common.CodeParamError, "参数错误")
		}
		return
	}

	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}
	slug := strings.TrimSpace(req.Slug)
	if pageSlugTaken(slug, page.ID) {
		common.Fail(c, common.CodeParamError, "slug 已存在")
		return
	}

	updates := map[string]any{
		"title":   strings.TrimSpace(req.Title),
		"slug":    slug,
		"content": req.Content,
		"status":  req.Status,
	}
	if err := model.DB.Model(&page).Updates(updates).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, page)
}

// AdminDeletePage DELETE /api/v1/admin/pages/:id
func AdminDeletePage(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}
	if err := model.DB.Delete(&page).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}
