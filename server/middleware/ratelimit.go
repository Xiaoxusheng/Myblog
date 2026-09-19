package middleware

import (
	"sync"
	"time"

	"myblog/server/common"

	"github.com/gin-gonic/gin"
)

// 游客评论限流：同 IP 60 秒内仅允许成功提交 1 次（契约 20003）
const commentWindow = 60 * time.Second

var (
	commentMu       sync.Mutex
	commentLastSeen = make(map[string]time.Time)
)

// CommentRateLimit 评论限流中间件：只做检查（命中直接 20003），
// 实际记录由 handler 在评论入库成功后调用 MarkComment——无效请求不占用限流窗口。
func CommentRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		commentMu.Lock()
		last, ok := commentLastSeen[c.ClientIP()]
		commentMu.Unlock()
		if ok && time.Since(last) < commentWindow {
			common.Fail(c, common.CodeTooFrequent, "操作过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}

// MarkComment 记录一次成功提交；顺手清理过期记录，避免 map 无界增长
func MarkComment(c *gin.Context) {
	commentMu.Lock()
	defer commentMu.Unlock()
	if len(commentLastSeen) > 1024 {
		now := time.Now()
		for ip, t := range commentLastSeen {
			if now.Sub(t) >= commentWindow {
				delete(commentLastSeen, ip)
			}
		}
	}
	commentLastSeen[c.ClientIP()] = time.Now()
}

// ResetCommentRateLimit 仅供测试用例之间隔离限流状态
func ResetCommentRateLimit() {
	commentMu.Lock()
	defer commentMu.Unlock()
	commentLastSeen = make(map[string]time.Time)
}
