package handler

// 管理端 URL 重定向管理（契约 #62-65）。环检测见 model.RedirectLoopExists。

import (
	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// redirectPayload 创建/更新重定向入参（契约 #63/#64）
type redirectPayload struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Type    int    `json:"type"`
	Enabled bool   `json:"enabled"`
}

// normalize 校验并规范化，非法时直接回 10001 并返回 false
func (p *redirectPayload) normalize(c *gin.Context) bool {
	p.Source = model.NormalizeRedirectPath(p.Source)
	p.Target = model.NormalizeRedirectPath(p.Target)
	switch {
	case p.Source == "" || p.Target == "":
		common.Fail(c, common.CodeParamError, "路径不能为空，且需以 / 开头")
	case p.Source == p.Target:
		common.Fail(c, common.CodeParamError, "来源与目标不能相同")
	case p.Type != model.RedirectPermanent && p.Type != model.RedirectTemporary:
		common.Fail(c, common.CodeParamError, "type 仅支持 301 / 302")
	default:
		return true
	}
	return false
}

// AdminListRedirects GET /api/v1/admin/redirects —— createdAt 倒序
func AdminListRedirects(c *gin.Context) {
	pq := common.ParsePage(c, 20)
	buildQuery := func() *gorm.DB {
		return model.DB.Model(&model.Redirect{})
	}
	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var list []model.Redirect
	if err := buildQuery().Order("created_at DESC, id DESC").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&list).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, pq.Data(list, total))
}

// AdminCreateRedirect POST /api/v1/admin/redirects
func AdminCreateRedirect(c *gin.Context) {
	var req redirectPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if !req.normalize(c) {
		return
	}
	var n int64
	model.DB.Model(&model.Redirect{}).Where("source = ?", req.Source).Count(&n)
	if n > 0 {
		common.Fail(c, common.CodeParamError, "该来源路径已存在重定向规则")
		return
	}
	if model.RedirectLoopExists(model.DB, req.Source, req.Target) {
		common.Fail(c, common.CodeParamError, "该重定向会形成循环跳转，已拒绝")
		return
	}
	rule := model.Redirect{Source: req.Source, Target: req.Target, Type: req.Type, Enabled: req.Enabled}
	if err := model.DB.Create(&rule).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"redirect": rule})
}

// AdminUpdateRedirect PUT /api/v1/admin/redirects/:id
func AdminUpdateRedirect(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "重定向 id 不合法")
		return
	}
	var rule model.Redirect
	if err := model.DB.First(&rule, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "重定向不存在")
		return
	}
	var req redirectPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if !req.normalize(c) {
		return
	}
	var n int64
	model.DB.Model(&model.Redirect{}).Where("source = ? AND id <> ?", req.Source, rule.ID).Count(&n)
	if n > 0 {
		common.Fail(c, common.CodeParamError, "该来源路径已存在重定向规则")
		return
	}
	if model.RedirectLoopExists(model.DB, req.Source, req.Target) {
		common.Fail(c, common.CodeParamError, "该重定向会形成循环跳转，已拒绝")
		return
	}
	updates := map[string]any{"source": req.Source, "target": req.Target, "type": req.Type, "enabled": req.Enabled}
	if err := model.DB.Model(&rule).Updates(updates).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"redirect": rule})
}

// AdminDeleteRedirect DELETE /api/v1/admin/redirects/:id
func AdminDeleteRedirect(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "重定向 id 不合法")
		return
	}
	var rule model.Redirect
	if err := model.DB.First(&rule, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "重定向不存在")
		return
	}
	if err := model.DB.Delete(&rule).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}
