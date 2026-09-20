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

// PostDetailDTO 详情 = PostSummary + content + updatedAt + SEO 扩展
type PostDetailDTO struct {
	PostSummaryDTO
	Content        string    `json:"content"`
	SeoTitle       string    `json:"seoTitle"`
	SeoDescription string    `json:"seoDescription"`
	Canonical      string    `json:"canonical"`
	OgImage        string    `json:"ogImage"`
	UpdatedAt      time.Time `json:"updatedAt"`
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

// PageDTO 页面完整对象（管理端详情/创建/更新/恢复，以及公开 #8 均返回此结构）
type PageDTO struct {
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Content        string     `json:"content"`
	Status         int8       `json:"status"`
	PageType       string     `json:"pageType"`
	PublishedAt    *time.Time `json:"publishedAt"`
	PublishAt      *time.Time `json:"publishAt"`
	SeoTitle       string     `json:"seoTitle"`
	SeoDescription string     `json:"seoDescription"`
	Canonical      string     `json:"canonical"`
	OgImage        string     `json:"ogImage"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// PageItemDTO 管理端列表项（不含 content，避免列表接口传大字段）
type PageItemDTO struct {
	ID          uint       `json:"id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Status      int8       `json:"status"`
	PageType    string     `json:"pageType"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	PublishedAt *time.Time `json:"publishedAt"`
	PublishAt   *time.Time `json:"publishAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// PageRevisionItemDTO 版本列表项（不含 content）
type PageRevisionItemDTO struct {
	ID        uint      `json:"id"`
	PageID    uint      `json:"pageId"`
	Version   int       `json:"version"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
}

// PageRevisionDetailDTO 版本详情 = 列表项 + 该版本全量快照
type PageRevisionDetailDTO struct {
	PageRevisionItemDTO
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
	Status  int8   `json:"status"`
}

// PageMetaDTO 列表页概览（按当前筛选条件的全量统计，不受分页影响）
type PageMetaDTO struct {
	TotalCount     int64 `json:"totalCount"`
	PublishedCount int64 `json:"publishedCount"`
	DraftCount     int64 `json:"draftCount"`
	ScheduledCount int64 `json:"scheduledCount"`
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
		SeoTitle:       p.SeoTitle,
		SeoDescription: p.SeoDescription,
		Canonical:      p.Canonical,
		OgImage:        p.OgImage,
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

// ToPageDTO 实体 → 完整页面 DTO
func ToPageDTO(p *Page) PageDTO {
	return PageDTO{
		ID:             p.ID,
		Title:          p.Title,
		Slug:           p.Slug,
		Content:        p.Content,
		Status:         p.Status,
		PageType:       p.PageType,
		PublishedAt:    p.PublishedAt,
		PublishAt:      p.PublishAt,
		SeoTitle:       p.SeoTitle,
		SeoDescription: p.SeoDescription,
		Canonical:      p.Canonical,
		OgImage:        p.OgImage,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

// ToPageItem 实体 → 管理端列表项
func ToPageItem(p *Page) PageItemDTO {
	return PageItemDTO{
		ID:          p.ID,
		Title:       p.Title,
		Slug:        p.Slug,
		Status:      p.Status,
		PageType:    p.PageType,
		UpdatedAt:   p.UpdatedAt,
		PublishedAt: p.PublishedAt,
		PublishAt:   p.PublishAt,
		CreatedAt:   p.CreatedAt,
	}
}

// ToPageRevisionItem 版本实体 → 列表项
func ToPageRevisionItem(r *PageRevision) PageRevisionItemDTO {
	return PageRevisionItemDTO{
		ID:        r.ID,
		PageID:    r.PageID,
		Version:   r.Version,
		Remark:    r.Remark,
		CreatedAt: r.CreatedAt,
	}
}

// ToPageRevisionDetail 版本实体 → 详情
func ToPageRevisionDetail(r *PageRevision) PageRevisionDetailDTO {
	return PageRevisionDetailDTO{
		PageRevisionItemDTO: ToPageRevisionItem(r),
		Title:               r.Title,
		Slug:                r.Slug,
		Content:             r.Content,
		Status:              r.Status,
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
