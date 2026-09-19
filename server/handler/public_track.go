package handler

// 公开埋点上报（契约 #66）。设计要点：
// - 静默：超限/非法输入一律 HTTP 200 {ok:false}，绝不给前台报错
// - 隐私：IP 只以哈希形态进入数据库（model.NewPageView）
// - 防滥用：同 IP 每分钟 60 条（内存滑动计数，进程级）

import (
	"strings"
	"sync"
	"time"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

const (
	trackWindow      = time.Minute
	trackLimitPerMin = 60
)

var (
	trackMu       sync.Mutex
	trackCounters = make(map[string]*trackCounter)
)

type trackCounter struct {
	windowStart time.Time
	count       int
}

// trackAllowed 内存限流：同 IP 每分钟 ≤ trackLimitPerMin 条
func trackAllowed(ip string) bool {
	trackMu.Lock()
	defer trackMu.Unlock()
	c, ok := trackCounters[ip]
	now := time.Now()
	if !ok || now.Sub(c.windowStart) >= trackWindow {
		if len(trackCounters) > 4096 { // 防无界增长：全量清理过期窗口
			for k, v := range trackCounters {
				if now.Sub(v.windowStart) >= trackWindow {
					delete(trackCounters, k)
				}
			}
		}
		trackCounters[ip] = &trackCounter{windowStart: now, count: 1}
		return true
	}
	if c.count >= trackLimitPerMin {
		return false
	}
	c.count++
	return true
}

// ResetTrackCounters 仅供测试用例之间隔离限流状态
func ResetTrackCounters() {
	trackMu.Lock()
	defer trackMu.Unlock()
	trackCounters = make(map[string]*trackCounter)
}

// Track POST /api/v1/track
func Track(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		PostID  uint   `json:"postId"`
		Referer string `json:"referer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.OK(c, gin.H{"ok": false})
		return
	}
	path := req.Path
	if path == "" || !strings.HasPrefix(path, "/") || len(path) > 512 {
		common.OK(c, gin.H{"ok": false})
		return
	}
	if !trackAllowed(c.ClientIP()) {
		common.OK(c, gin.H{"ok": false})
		return
	}

	// postId 合法性校验：非法值按非文章页记录
	postID := req.PostID
	if postID > 0 {
		var n int64
		model.DB.Model(&model.Post{}).Where("id = ?", postID).Count(&n)
		if n == 0 {
			postID = 0
		}
	}

	pv := model.NewPageView(path, postID, c.Request.UserAgent(), req.Referer, c.ClientIP())
	if err := model.DB.Create(&pv).Error; err != nil {
		// 上报失败对前台无感
		common.OK(c, gin.H{"ok": false})
		return
	}
	common.OK(c, gin.H{"ok": true})
}
