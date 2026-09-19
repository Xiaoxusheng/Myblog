package handler

import (
	"strings"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// ---------- 友链（契约 #34-37） ----------

// AdminListLinks GET /api/v1/admin/links —— 按 sort 升序
func AdminListLinks(c *gin.Context) {
	pq := common.ParsePage(c, 10)

	var total int64
	if err := model.DB.Model(&model.Link{}).Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var links []model.Link
	if err := model.DB.Order("sort ASC, id ASC").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&links).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, pq.Data(links, total))
}

// linkPayload 创建/更新友链入参
type linkPayload struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Logo        string `json:"logo"`
	Description string `json:"description"`
	Visible     bool   `json:"visible"`
	Sort        int    `json:"sort"`
}

func (p *linkPayload) validate(c *gin.Context) bool {
	switch {
	case strings.TrimSpace(p.Name) == "":
		common.Fail(c, common.CodeParamError, "名称不能为空")
	case strings.TrimSpace(p.URL) == "":
		common.Fail(c, common.CodeParamError, "URL 不能为空")
	case utf8.RuneCountInString(strings.TrimSpace(p.Name)) > 100:
		common.Fail(c, common.CodeParamError, "名称不能超过 100 字")
	case utf8.RuneCountInString(strings.TrimSpace(p.URL)) > 500 ||
		utf8.RuneCountInString(strings.TrimSpace(p.Logo)) > 512 ||
		utf8.RuneCountInString(strings.TrimSpace(p.Description)) > 500:
		common.Fail(c, common.CodeParamError, "字段长度超出限制")
	default:
		return true
	}
	return false
}

// AdminCreateLink POST /api/v1/admin/links
func AdminCreateLink(c *gin.Context) {
	var req linkPayload
	if err := c.ShouldBindJSON(&req); err != nil || !req.validate(c) {
		if err != nil {
			common.Fail(c, common.CodeParamError, "参数错误")
		}
		return
	}

	link := model.Link{
		Name:        strings.TrimSpace(req.Name),
		URL:         strings.TrimSpace(req.URL),
		Logo:        strings.TrimSpace(req.Logo),
		Description: strings.TrimSpace(req.Description),
		Visible:     req.Visible,
		Sort:        req.Sort,
	}
	if err := model.DB.Create(&link).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, link)
}

// AdminUpdateLink PUT /api/v1/admin/links/:id —— 全量更新
func AdminUpdateLink(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req linkPayload
	if err := c.ShouldBindJSON(&req); err != nil || !req.validate(c) {
		if err != nil {
			common.Fail(c, common.CodeParamError, "参数错误")
		}
		return
	}

	var link model.Link
	if err := model.DB.First(&link, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "友链不存在")
		return
	}

	updates := map[string]any{
		"name":        strings.TrimSpace(req.Name),
		"url":         strings.TrimSpace(req.URL),
		"logo":        strings.TrimSpace(req.Logo),
		"description": strings.TrimSpace(req.Description),
		"visible":     req.Visible,
		"sort":        req.Sort,
	}
	if err := model.DB.Model(&link).Updates(updates).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, link)
}

// AdminDeleteLink DELETE /api/v1/admin/links/:id
func AdminDeleteLink(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var link model.Link
	if err := model.DB.First(&link, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "友链不存在")
		return
	}
	if err := model.DB.Delete(&link).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}
