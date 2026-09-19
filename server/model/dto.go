package model

import "time"

// ---------- 契约 DTO（响应字段与 docs/contracts/api.md 一字不差） ----------

// CategoryRefDTO PostSummary 内嵌的分类引用 {id,name,slug}
type CategoryRefDTO struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// TagRefDTO PostSummary 内嵌的标签引用 {id,name,slug}
type TagRefDTO struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// PostSummaryDTO 列表项（不含 content）
type PostSummaryDTO struct {
	ID          uint            `json:"id"`
	Title       string          `json:"title"`
	Slug        string          `json:"slug"`
	Summary     string          `json:"summary"`
	Cover       string          `json:"cover"`
	ViewCount   int             `json:"viewCount"`
	LikeCount   int             `json:"likeCount"`
	Status      int8            `json:"status"`
	IsTop       bool            `json:"isTop"`
	CreatedAt   time.Time       `json:"createdAt"`
	PublishedAt *time.Time      `json:"publishedAt"`
	Category    *CategoryRefDTO `json:"category"`
	Tags        []TagRefDTO     `json:"tags"`
}

// PostDetailDTO 详情 = PostSummary + content + updatedAt
type PostDetailDTO struct {
	PostSummaryDTO
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AdminPostItemDTO = PostSummary + content + categoryId + tagNames + commentCount + publishAt + series
type AdminPostItemDTO struct {
	PostSummaryDTO
	Content        string     `json:"content"`
	CategoryID     uint       `json:"categoryId"`
	TagNames       []string   `json:"tagNames"`
	CommentCount   int64      `json:"commentCount"`
	PublishAt      *time.Time `json:"publishAt"` // 仅定时发布有值
	SeriesID       uint       `json:"seriesId"`  // 0=不属于专题
	SeriesSort     int        `json:"seriesSort"`
	SeoTitle       string     `json:"seoTitle"`
	SeoDescription string     `json:"seoDescription"`
	Canonical      string     `json:"canonical"`
	OgImage        string     `json:"ogImage"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// PostRefDTO 上一篇/下一篇引用 {id,title,slug}|null
type PostRefDTO struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

// ---------- 映射 ----------

func toCategoryRef(c *Category) *CategoryRefDTO {
	if c == nil {
		return nil
	}
	return &CategoryRefDTO{ID: c.ID, Name: c.Name, Slug: c.Slug}
}

func toTagRefs(tags []Tag) []TagRefDTO {
	refs := make([]TagRefDTO, 0, len(tags))
	for i := range tags {
		refs = append(refs, TagRefDTO{ID: tags[i].ID, Name: tags[i].Name, Slug: tags[i].Slug})
	}
	return refs
}

// ToPostSummary 实体 → 列表项 DTO
func ToPostSummary(p *Post) PostSummaryDTO {
	return PostSummaryDTO{
		ID:          p.ID,
		Title:       p.Title,
		Slug:        p.Slug,
		Summary:     p.Summary,
		Cover:       p.Cover,
		ViewCount:   p.ViewCount,
		LikeCount:   p.LikeCount,
		Status:      p.Status,
		IsTop:       p.IsTop,
		CreatedAt:   p.CreatedAt,
		PublishedAt: p.PublishedAt,
		Category:    toCategoryRef(p.Category),
		Tags:        toTagRefs(p.Tags),
	}
}

// ToPostDetail 实体 → 详情 DTO
func ToPostDetail(p *Post) PostDetailDTO {
	return PostDetailDTO{
		PostSummaryDTO: ToPostSummary(p),
		Content:        p.Content,
		UpdatedAt:      p.UpdatedAt,
	}
}

// ToAdminPostItem 实体 → 管理端文章对象
func ToAdminPostItem(p *Post, commentCount int64) AdminPostItemDTO {
	names := make([]string, 0, len(p.Tags))
	for i := range p.Tags {
		names = append(names, p.Tags[i].Name)
	}
	return AdminPostItemDTO{
		PostSummaryDTO: ToPostSummary(p),
		Content:        p.Content,
		CategoryID:     p.CategoryID,
		TagNames:       names,
		CommentCount:   commentCount,
		PublishAt:      p.PublishAt,
		SeriesID:       p.SeriesID,
		SeriesSort:     p.SeriesSort,
		SeoTitle:       p.SeoTitle,
		SeoDescription: p.SeoDescription,
		Canonical:      p.Canonical,
		OgImage:        p.OgImage,
		UpdatedAt:      p.UpdatedAt,
	}
}

// ToCommentAdmin 实体 → 管理端评论 DTO
func ToCommentAdmin(cm *Comment, postTitle string) CommentAdminDTO {
	return CommentAdminDTO{
		ID:        cm.ID,
		PostID:    cm.PostID,
		PostTitle: postTitle,
		ParentID:  cm.ParentID,
		Nickname:  cm.Nickname,
		Email:     cm.Email,
		Website:   cm.Website,
		Content:   cm.Content,
		Status:    cm.Status,
		IP:        cm.IP,
		IsAdmin:   cm.IsAdmin,
		CreatedAt: cm.CreatedAt,
	}
}
