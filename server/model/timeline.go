package model

import "time"

// TimelineEvent 技术时间线节点（契约 #91-94 / #99）
type TimelineEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"type:varchar(100)" json:"title"`
	Content     string    `gorm:"type:text" json:"content"` // Markdown 原文
	EventDate   time.Time `json:"eventDate"`
	Image       string    `gorm:"type:varchar(512)" json:"image"`
	PostID      uint      `gorm:"index" json:"postId"` // 0=无关联；弱关联不设外键
	ProjectName string    `gorm:"type:varchar(100)" json:"projectName"`
	ProjectURL  string    `gorm:"type:varchar(512)" json:"projectUrl"`
	Visible     bool      `json:"visible"` // 无 default 标签，规避 GORM 零值坑（DECISIONS #17）
	Sort        int       `json:"sort"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
