package model

// 评论防护黑名单与后台通知中心（契约 #69-75）。

import (
	"time"

	"gorm.io/gorm"
)

// CommentBlacklist 评论黑名单（命中 → 评论直接标记垃圾）
type CommentBlacklist struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"uniqueIndex:idx_blacklist_type_value;type:varchar(16)" json:"type"` // ip/email/keyword
	Value     string    `gorm:"uniqueIndex:idx_blacklist_type_value;type:varchar(200)" json:"value"`
	CreatedAt time.Time `json:"createdAt"`
}

// Notification 后台通知（铃铛轮询，无 WebSocket）
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"type:varchar(32)" json:"type"` // comment_pending/comment_spam/post_published/backup
	Title     string    `gorm:"type:varchar(200)" json:"title"`
	Content   string    `gorm:"type:varchar(500)" json:"content"`
	Link      string    `gorm:"type:varchar(500)" json:"link"` // 管理端跳转路径
	Read      bool      `gorm:"index" json:"read"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateNotification 写入一条通知；失败只返回错误由调用方决定是否忽略（通知不应阻断业务）
func CreateNotification(db *gorm.DB, nType, title, content, link string) error {
	n := Notification{Type: nType, Title: title, Content: content, Link: link}
	if len([]rune(content)) > 500 {
		n.Content = string([]rune(content)[:500])
	}
	return db.Create(&n).Error
}
