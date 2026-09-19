package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 跨域放开（前后端分离 + 管理端独立域名部署）
func CORS() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:    []string{"Authorization", "Content-Type"},
		ExposeHeaders:   []string{"Content-Length"},
		// AllowAllOrigins 与 AllowCredentials=true 不兼容（gin-contrib/cors 会 panic）
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}
