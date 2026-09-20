package middleware

import (
	"strings"
	"time"

	"myblog/server/common"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWT 上下文键
const (
	ContextUserID       = "authUserID"
	ContextUsername     = "authUsername"
	ContextTokenVersion = "authTokenVersion"
)

// JWT 鉴权中间件：仅 /admin/* 路由组挂载。
// 失败统一 HTTP 401 + code 10002（前端跳登录）。
// token 中的 ver（签发时的 token_version）由 handler.currentUser 与 DB 比对，实现改密吊销。
func JWT(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			common.Unauthorized(c)
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			return secret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || token == nil || !token.Valid {
			common.Unauthorized(c)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			common.Unauthorized(c)
			return
		}
		uid, ok := claims["uid"].(float64)
		if !ok || uid <= 0 {
			common.Unauthorized(c)
			return
		}
		c.Set(ContextUserID, uint(uid))
		if username, ok := claims["username"].(string); ok {
			c.Set(ContextUsername, username)
		}
		if ver, ok := claims["ver"].(float64); ok {
			c.Set(ContextTokenVersion, int(ver))
		}
		c.Next()
	}
}

// GenerateToken 签发 HS256 JWT；tokenVersion 随 claims 下发（ver），供改密后吊销校验
func GenerateToken(userID uint, username, secret string, ttl time.Duration, tokenVersion int) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"uid":      userID,
		"username": username,
		"ver":      tokenVersion,
		"iat":      now.Unix(),
		"exp":      now.Add(ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
