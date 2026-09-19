// Package common 提供统一响应、错误码与分页解析。
package common

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应 body：{"code":0,"message":"ok","data":...}
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 业务成功（HTTP 200）
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: CodeOK, Message: "ok", Data: data})
}

// Fail 业务失败（HTTP 200 + 错误码）
func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{Code: code, Message: message, Data: nil})
}

// Unauthorized 鉴权失败：HTTP 401（前端统一跳登录）
func Unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized,
		Response{Code: CodeUnauthorized, Message: "未登录或登录已过期", Data: nil})
}

// ServerError 服务器内部错误：HTTP 500，日志记录根因但不泄露给客户端
func ServerError(c *gin.Context, err error) {
	if err != nil {
		log.Printf("[server-error] %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	}
	c.JSON(http.StatusInternalServerError,
		Response{Code: http.StatusInternalServerError, Message: "服务器内部错误", Data: nil})
}
