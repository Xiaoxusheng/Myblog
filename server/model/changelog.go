package model

import "time"

// Changelog 版本发布记录（契约 #95-98 / #100）
type Changelog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Version    string    `gorm:"type:varchar(32);uniqueIndex" json:"version"`
	Title      string    `gorm:"type:varchar(100)" json:"title"`
	Content    string    `gorm:"type:text" json:"content"` // Markdown 原文
	ReleasedAt time.Time `json:"releasedAt"`
	Status     int8      `json:"status"` // 0 草稿 1 已发布
	Sort       int       `json:"sort"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// ChangelogDraft / ChangelogPublished 状态枚举
const (
	ChangelogDraft     int8 = 0
	ChangelogPublished int8 = 1
)
