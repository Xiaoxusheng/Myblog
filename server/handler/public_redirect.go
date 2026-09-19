package handler

// 公开重定向解析（契约 #55）：前台 404 兜底路由先询问本接口，命中则跳转目标路径

import (
	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// ResolveRedirect GET /api/v1/redirects/resolve?path=/post/old
// 始终返回 code 0；命中 enabled 规则时 data.redirect 非空
func ResolveRedirect(c *gin.Context) {
	path := model.NormalizeRedirectPath(c.Query("path"))
	if path == "" {
		common.OK(c, gin.H{"redirect": nil})
		return
	}
	var rule model.Redirect
	err := model.DB.Where("source = ? AND enabled = ?", path, true).First(&rule).Error
	if err != nil {
		common.OK(c, gin.H{"redirect": nil})
		return
	}
	common.OK(c, gin.H{"redirect": gin.H{
		"source": rule.Source,
		"target": rule.Target,
		"type":   rule.Type,
	}})
}
