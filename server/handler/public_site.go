package handler

import (
	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// GetSite GET /api/v1/site —— 站点信息：Settings + categories + tags（含已发布文章数）
func GetSite(c *gin.Context) {
	settings := getSettingsDTO(model.DB)

	var categories []model.Category
	if err := model.DB.Order("id ASC").Find(&categories).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	fillCategoryPostCounts(categories, true)

	var tags []model.Tag
	if err := model.DB.Order("id ASC").Find(&tags).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	fillTagPostCounts(tags, true)

	common.OK(c, gin.H{
		"settings":   settings,
		"categories": categories,
		"tags":       tags,
	})
}

// ---------- 文章数统计（分类/标签页 & 站点信息共用） ----------

type countRow struct {
	Key uint  `gorm:"column:k"`
	Cnt int64 `gorm:"column:cnt"`
}

func fillCategoryPostCounts(categories []model.Category, publishedOnly bool) {
	db := model.DB.Model(&model.Post{}).
		Select("category_id AS k, COUNT(*) AS cnt").Group("category_id")
	if publishedOnly {
		db = db.Where("status = ?", model.PostPublished)
	}
	var rows []countRow
	if err := db.Scan(&rows).Error; err != nil {
		return
	}
	m := make(map[uint]int64, len(rows))
	for _, r := range rows {
		m[r.Key] = r.Cnt
	}
	for i := range categories {
		categories[i].PostCount = m[categories[i].ID]
	}
}

func fillTagPostCounts(tags []model.Tag, publishedOnly bool) {
	q := model.DB.Table("post_tags").
		Select("post_tags.tag_id AS k, COUNT(*) AS cnt").
		Joins("JOIN posts ON posts.id = post_tags.post_id")
	if publishedOnly {
		q = q.Where("posts.status = ?", model.PostPublished)
	}
	q = q.Group("post_tags.tag_id")
	var rows []countRow
	if err := q.Scan(&rows).Error; err != nil {
		return
	}
	m := make(map[uint]int64, len(rows))
	for _, r := range rows {
		m[r.Key] = r.Cnt
	}
	for i := range tags {
		tags[i].PostCount = m[tags[i].ID]
	}
}
