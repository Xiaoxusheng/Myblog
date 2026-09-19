package model

import "time"

// 文章状态（契约枚举）
const (
	PostDraft     int8 = 0 // 草稿
	PostPublished int8 = 1 // 已发布
	PostHidden    int8 = 2 // 隐藏
	PostScheduled int8 = 3 // 定时发布（到点由调度器自动置为已发布）
)

// Post 文章
type Post struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	Title          string     `gorm:"type:varchar(200)" json:"title"`
	Slug           string     `gorm:"type:varchar(200);uniqueIndex" json:"slug"`
	Summary        string     `gorm:"type:varchar(1000)" json:"summary"`
	Content        string     `gorm:"type:text" json:"content"`
	Cover          string     `gorm:"type:varchar(512)" json:"cover"`
	CategoryID     uint       `gorm:"index" json:"categoryId"` // 0=未分类
	ViewCount      int        `gorm:"default:0" json:"viewCount"`
	LikeCount      int        `gorm:"default:0" json:"likeCount"`
	Status         int8       `gorm:"index;default:0" json:"status"`
	IsTop          bool       `json:"isTop"`
	PublishedAt    *time.Time `gorm:"index" json:"publishedAt"`        // 首次置为已发布时写入；列表/RSS/sitemap 按此排序
	PublishAt      *time.Time `gorm:"index" json:"publishAt"`          // 仅 status=3 有值：计划发布时间；调度器按到期扫描
	SeriesID       uint       `gorm:"index;default:0" json:"seriesId"` // 0=不属于专题（一文至多一专题）
	SeriesSort     int        `gorm:"default:0" json:"seriesSort"`     // 专题内序号，升序，同序按 id
	SeoTitle       string     `gorm:"type:varchar(200)" json:"seoTitle"`
	SeoDescription string     `gorm:"type:varchar(300)" json:"seoDescription"`
	Canonical      string     `gorm:"type:varchar(512)" json:"canonical"`
	OgImage        string     `gorm:"type:varchar(512)" json:"ogImage"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`

	// 关联仅用于 Preload / 级联，序列化统一走 DTO
	Category *Category `gorm:"foreignKey:CategoryID" json:"-"`
	Tags     []Tag     `gorm:"many2many:post_tags" json:"-"`
}
