package handler

// 评论垃圾检测（契约 #5 / 十、评论 Spam 系统）：轻量规则，无第三方服务。
// 注意：Comment.Email/Comment.IP 为加密列，不能参与 SQL 等值查询，
// 黑名单比对在 Go 侧内存进行；content/nickname 为明文列可直接查。

import (
	"fmt"
	"strings"
	"time"

	"myblog/server/model"

	"gorm.io/gorm"
)

const (
	spamLinkThreshold    = 3 // 内容中 URL 数达到该值判为垃圾
	spamNicknameBurst    = 3 // 同昵称窗口期内的待审/垃圾条数上限
	spamNicknameBurstWin = time.Minute * 10
)

// isSpam 依次执行规则，命中即返回 true：
// IP/邮箱黑名单、关键词黑名单、同文重复、URL 过多、同昵称高频。
func isSpam(db *gorm.DB, postID uint, nickname, email, content, ip string) bool {
	// 1/2. 黑名单（内存比对，规避加密列不可查询）
	var blacklist []model.CommentBlacklist
	if err := db.Find(&blacklist).Error; err != nil {
		blacklist = nil // 读失败时退化为不检查黑名单，不阻断评论
	}
	for _, b := range blacklist {
		switch b.Type {
		case "ip":
			if b.Value == ip {
				return true
			}
		case "email":
			if strings.EqualFold(b.Value, email) {
				return true
			}
		case "keyword":
			if b.Value != "" && strings.Contains(strings.ToLower(content), strings.ToLower(b.Value)) {
				return true
			}
		}
	}

	// 3. 相同内容重复提交（同文章下任意可见/待审/垃圾状态已存在）
	var dup int64
	if err := db.Model(&model.Comment{}).
		Where("post_id = ? AND content = ? AND status <> ?", postID, content, model.CommentRejected).
		Count(&dup).Error; err == nil && dup > 0 {
		return true
	}

	// 4. 链接过多
	if countURLs(content) >= spamLinkThreshold {
		return true
	}

	// 5. 同昵称高频（10 分钟内 ≥3 条待审/垃圾）
	var burst int64
	if err := db.Model(&model.Comment{}).
		Where("nickname = ? AND status IN ? AND created_at > ?",
			nickname, []int8{model.CommentPending, model.CommentSpam},
			time.Now().Add(-spamNicknameBurstWin)).
		Count(&burst).Error; err == nil && burst+1 >= int64(spamNicknameBurst) {
		return true
	}
	return false
}

// countURLs 统计内容中的 http(s) 链接数
func countURLs(content string) int {
	return strings.Count(content, "http://") + strings.Count(content, "https://")
}

// notifyCommentCreated 评论入库后的通知（垃圾 → comment_spam，待审 → comment_pending）
func notifyCommentCreated(db *gorm.DB, postTitle, content string, spam bool) {
	preview := content
	if r := []rune(preview); len(r) > 50 {
		preview = string(r[:50]) + "…"
	}
	if spam {
		_ = model.CreateNotification(db, "comment_spam", "捕获垃圾评论",
			fmt.Sprintf("《%s》：%s", postTitle, preview), "/comments?status=3")
		return
	}
	_ = model.CreateNotification(db, "comment_pending", "新评论待审核",
		fmt.Sprintf("《%s》：%s", postTitle, preview), "/comments?status=0")
}
