package handler

import (
	"strings"
	"time"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListPosts GET /api/v1/posts —— 公开文章分页：仅已发布、置顶优先、keyword/categoryId/tagId 过滤、sort 排序
func ListPosts(c *gin.Context) {
	pq := common.ParsePage(c, 10)

	// 先解析过滤参数（非法值直接 10001，避免在闭包里处理错误）
	sortParam := c.Query("sort")
	if sortParam != "" && sortParam != "newest" && sortParam != "views" && sortParam != "likes" {
		common.Fail(c, common.CodeParamError, "sort 仅支持 newest / views / likes")
		return
	}
	tagID := queryUint(c, "tagId")

	buildQuery := func() *gorm.DB {
		db := model.DB.Model(&model.Post{}).Where("status = ?", model.PostPublished)
		if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
			like := likeContains(kw)
			db = db.Where(
				"(title "+likeEscapeClause+" OR summary "+likeEscapeClause+" OR content "+likeEscapeClause+")",
				like, like, like,
			)
		}
		if catID := queryUint(c, "categoryId"); catID > 0 {
			db = db.Where("category_id = ?", catID)
		}
		if tagID > 0 {
			db = db.Joins("JOIN post_tags ON post_tags.post_id = posts.id AND post_tags.tag_id = ?", tagID)
		}
		switch sortParam {
		case "views":
			db = db.Order("is_top DESC, view_count DESC, id DESC")
		case "likes":
			db = db.Order("is_top DESC, like_count DESC, id DESC")
		default: // newest
			db = db.Order("is_top DESC, published_at DESC, id DESC")
		}
		return db
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var posts []model.Post
	if err := buildQuery().Preload("Category").Preload("Tags").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&posts).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	list := make([]model.PostSummaryDTO, 0, len(posts))
	for i := range posts {
		list = append(list, model.ToPostSummary(&posts[i]))
	}
	common.OK(c, pq.Data(list, total))
}

// GetPost GET /api/v1/posts/:slug —— 详情（id 或 slug），浏览量+1，附 prev/next/related
func GetPost(c *gin.Context) {
	post, found, err := resolvePublishedPost(c.Param("slug"))
	if err != nil {
		common.ServerError(c, err)
		return
	}
	if !found {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}

	// 浏览量 +1（UpdateColumn 不触发 updated_at）
	if err := model.DB.Model(post).UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	post.ViewCount++

	seriesInfo, seriesPrev, seriesNext := seriesContext(post)

	common.OK(c, gin.H{
		"post":       model.ToPostDetail(post),
		"prev":       adjacentPost(post, false),
		"next":       adjacentPost(post, true),
		"related":    relatedPosts(post),
		"series":     seriesInfo,
		"seriesPrev": seriesPrev,
		"seriesNext": seriesNext,
	})
}

// seriesContext 文章所属专题上下文（契约：series:{id,name,slug,index,total} + 相邻文章）。
// 不属于专题或专题未展示时返回 nil，前端不渲染专题区块。
func seriesContext(post *model.Post) (series any, prev, next *model.PostRefDTO) {
	if post.SeriesID == 0 {
		return nil, nil, nil
	}
	var seriesRow model.Series
	if err := model.DB.Where("id = ? AND visible = ?", post.SeriesID, true).
		First(&seriesRow).Error; err != nil {
		return nil, nil, nil
	}
	var siblings []model.Post
	if err := model.DB.Where("series_id = ? AND status = ?", post.SeriesID, model.PostPublished).
		Order("series_sort ASC, id ASC").Find(&siblings).Error; err != nil {
		return nil, nil, nil
	}
	index := -1
	for i := range siblings {
		if siblings[i].ID == post.ID {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, nil, nil
	}
	ref := func(p *model.Post) *model.PostRefDTO {
		return &model.PostRefDTO{ID: p.ID, Title: p.Title, Slug: p.Slug}
	}
	info := gin.H{
		"id":    seriesRow.ID,
		"name":  seriesRow.Name,
		"slug":  seriesRow.Slug,
		"index": index + 1,
		"total": len(siblings),
	}
	if index > 0 {
		prev = ref(&siblings[index-1])
	}
	if index < len(siblings)-1 {
		next = ref(&siblings[index+1])
	}
	return info, prev, next
}

// adjacentPost 相邻文章：prev=较早发布（newer=false 取更早），next=较晚发布。
// 以 (published_at, id) 双键比较，保证同秒发布时顺序稳定。
func adjacentPost(post *model.Post, newer bool) *model.PostRefDTO {
	var t time.Time
	if post.PublishedAt != nil {
		t = *post.PublishedAt
	}
	var p model.Post
	var err error
	if newer {
		err = model.DB.Where(
			"status = ? AND (published_at > ? OR (published_at = ? AND id > ?))",
			model.PostPublished, t, t, post.ID,
		).Order("published_at ASC, id ASC").First(&p).Error
	} else {
		err = model.DB.Where(
			"status = ? AND (published_at < ? OR (published_at = ? AND id < ?))",
			model.PostPublished, t, t, post.ID,
		).Order("published_at DESC, id DESC").First(&p).Error
	}
	if err != nil {
		return nil
	}
	return &model.PostRefDTO{ID: p.ID, Title: p.Title, Slug: p.Slug}
}

// relatedPosts 相关文章 = 同分类取 5（排除自身）
func relatedPosts(post *model.Post) []model.PostSummaryDTO {
	var posts []model.Post
	if err := model.DB.Preload("Category").Preload("Tags").
		Where("status = ? AND category_id = ? AND id <> ?", model.PostPublished, post.CategoryID, post.ID).
		Order("published_at DESC, id DESC").Limit(5).Find(&posts).Error; err != nil {
		return []model.PostSummaryDTO{}
	}
	list := make([]model.PostSummaryDTO, 0, len(posts))
	for i := range posts {
		list = append(list, model.ToPostSummary(&posts[i]))
	}
	return list
}

// LikePost GET... POST /api/v1/posts/:slug/like —— likeCount +1
func LikePost(c *gin.Context) {
	post, found, err := resolvePublishedPost(c.Param("slug"))
	if err != nil {
		common.ServerError(c, err)
		return
	}
	if !found {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}
	if err := model.DB.Model(post).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	post.LikeCount++
	common.OK(c, gin.H{"likeCount": post.LikeCount})
}
