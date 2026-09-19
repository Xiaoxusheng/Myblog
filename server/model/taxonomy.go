package model

import "time"

// Category 分类
type Category struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(64);uniqueIndex" json:"name"`
	Slug        string    `gorm:"type:varchar(64)" json:"slug"`
	Description string    `gorm:"type:varchar(500)" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	PostCount int64 `gorm:"-" json:"postCount"` // 由查询填充，不落库
}

// Tag 标签
type Tag struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(64);uniqueIndex" json:"name"`
	Slug      string    `gorm:"type:varchar(64)" json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	PostCount int64 `gorm:"-" json:"postCount"` // 由查询填充，不落库
}

// PostTag 文章-标签关联（复合主键 post_id+tag_id，tag_id 单独建索引）
type PostTag struct {
	PostID uint `gorm:"primaryKey;autoIncrement:false" json:"postId"`
	TagID  uint `gorm:"primaryKey;autoIncrement:false;index" json:"tagId"`
}
