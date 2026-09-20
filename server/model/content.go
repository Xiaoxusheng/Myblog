package model

import "time"

// Link 友情链接
type Link struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(100)" json:"name"`
	URL         string    `gorm:"type:varchar(500)" json:"url"`
	Logo        string    `gorm:"type:varchar(512)" json:"logo"`
	Description string    `gorm:"type:varchar(500)" json:"description"`
	Visible     bool      `json:"visible"`
	Sort        int       `json:"sort"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Page 自定义页面（关于等）
type Page struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"type:varchar(200)" json:"title"`
	Slug      string    `gorm:"type:varchar(200);uniqueIndex" json:"slug"`
	Content   string    `gorm:"type:text" json:"content"`
	Status    int8      `json:"status"` // 0 草稿 1 已发布
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Setting 站点设置（key-value，key 为主键；bool/number 转字符串存取）
type Setting struct {
	// 列名显式声明：key 是 MySQL 8 保留字，不显式声明时 GORM 生成的部分语句
	// 会裸写 `key` 触发 Error 1064（SQLite 不保留 key，本地测试发现不了）
	Key       string    `gorm:"column:key;type:varchar(64);primaryKey" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SettingsDTO 契约 Settings 结构化对象
type SettingsDTO struct {
	SiteName        string `json:"siteName"`
	SiteDescription string `json:"siteDescription"`
	SiteKeywords    string `json:"siteKeywords"`
	SiteURL         string `json:"siteUrl"`
	Logo            string `json:"logo"`
	Notice          string `json:"notice"`
	ICP             string `json:"icp"`
	FooterText      string `json:"footerText"`
	CommentEnabled  bool   `json:"commentEnabled"`
	PostPageSize    int    `json:"postPageSize"`
	// AutoRedirectOnSlugChange 文章 slug 变更时自动创建旧→新 301 重定向（默认开启）
	AutoRedirectOnSlugChange bool `json:"autoRedirectOnSlugChange"`
}

// Upload 上传媒体
type Upload struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Filename  string    `gorm:"type:varchar(255)" json:"filename"`
	Path      string    `gorm:"type:varchar(512)" json:"-"` // 相对 UploadDir，如 202609/xxx.png
	URL       string    `gorm:"type:varchar(512)" json:"url"`
	Size      int64     `json:"size"`
	Mime      string    `gorm:"type:varchar(100)" json:"mime"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
