package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 跨域策略：origins 为空时放开所有 Origin（前后端分离历史行为，
// 凭据=false，不含 Cookie）；配置 BLOG_CORS_ORIGINS 后仅放行白名单 Origin。
func CORS(origins []string) gin.HandlerFunc {
	base := cors.Config{
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:  []string{"Authorization", "Content-Type"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}
	if len(origins) == 0 {
		// AllowAllOrigins 与 AllowCredentials=true 不兼容（gin-contrib/cors 会 panic）
		base.AllowAllOrigins = true
		base.AllowCredentials = false
	} else {
		base.AllowOrigins = origins
	}
	return cors.New(base)
}
