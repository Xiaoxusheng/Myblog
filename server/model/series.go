package model

// Series 专题（契约：教程型系列文章的内容组织；成员关系存 posts.series_id / series_sort）

import "time"

type Series struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(100);uniqueIndex" json:"name"`
	Slug        string `gorm:"type:varchar(100);uniqueIndex" json:"slug"`
	Description string `gorm:"type:varchar(500)" json:"description"`
	Cover       string `gorm:"type:varchar(512)" json:"cover"`
	// Visible 不加 default 标签：GORM 会对带默认值的零值字段做插入省略，false 会被写成 true
	Visible   bool      `json:"visible"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
