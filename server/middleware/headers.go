package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders 基础安全响应头（本服务承载 API、RSS 与静态上传文件）：
// 禁止 MIME 嗅探、禁止被 iframe 嵌套、限制 Referer 泄漏。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
