package middleware

import (
	"sync"
	"time"

	"myblog/server/common"

	"github.com/gin-gonic/gin"
)

// 登录防爆破：同一 IP 连续失败达 loginMaxFailures 次后锁定 loginLockWindow；
// 成功登录立即解除；距最后一次失败超过窗口后自动解锁。
const (
	loginMaxFailures = 5
	loginLockWindow  = 15 * time.Minute
)

type loginState struct {
	failures int
	lastFail time.Time
}

var (
	loginMu       sync.Mutex
	loginFailures = make(map[string]*loginState)
)

// LoginRateLimit 登录限流中间件：锁定期内直接 20003，请求不进入 handler。
// 锁定期内仍持续尝试视为恶意爆破，向防护 Guard 计点，累积达阈值自动封禁该 IP。
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		loginMu.Lock()
		st, ok := loginFailures[ip]
		blocked := ok && st.failures >= loginMaxFailures && time.Since(st.lastFail) < loginLockWindow
		loginMu.Unlock()
		if blocked {
			if guard != nil {
				guard.addEvent(ip, eventLoginBlock, "登录锁定期间继续尝试")
				guard.addStrike(ip, "登录锁定期间持续尝试登录")
			}
			common.Fail(c, common.CodeTooFrequent, "登录失败次数过多，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}

// MarkLoginFailure 记录一次登录失败（handler 校验不通过时调用）
func MarkLoginFailure(c *gin.Context) {
	loginMu.Lock()
	defer loginMu.Unlock()
	if len(loginFailures) > 4096 {
		now := time.Now()
		for ip, st := range loginFailures {
			if now.Sub(st.lastFail) >= loginLockWindow {
				delete(loginFailures, ip)
			}
		}
	}
	ip := c.ClientIP()
	st, ok := loginFailures[ip]
	if !ok || time.Since(st.lastFail) >= loginLockWindow {
		st = &loginState{}
		loginFailures[ip] = st
	}
	st.failures++
	st.lastFail = time.Now()
}

// MarkLoginSuccess 登录成功后清除失败记录
func MarkLoginSuccess(c *gin.Context) {
	loginMu.Lock()
	defer loginMu.Unlock()
	delete(loginFailures, c.ClientIP())
}

// ResetLoginRateLimit 仅供测试用例之间隔离限流状态
func ResetLoginRateLimit() {
	loginMu.Lock()
	defer loginMu.Unlock()
	loginFailures = make(map[string]*loginState)
}
