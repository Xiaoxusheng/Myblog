package handler

// 公开专题接口（契约 #53/#54）：仅 visible 专题与已发布文章

import (
	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// publishedStatus 供 seriesCountMap 过滤已发布文章数
func publishedStatus() *int8 {
	s := model.PostPublished
	return &s
}

// ListSeries GET /api/v1/series —— visible 专题，sort 升序，postCount=已发布文章数
func ListSeries(c *gin.Context) {
	var list []model.Series
	if err := model.DB.Where("visible = ?", true).
		Order("sort ASC, id ASC").Find(&list).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	counts := seriesCountMap(publishedStatus())
	dtos := make([]seriesDTO, 0, len(list))
	for i := range list {
		dtos = append(dtos, toSeriesDTO(&list[i], counts[list[i].ID]))
	}
	common.OK(c, gin.H{"list": dtos})
}

// GetSeries GET /api/v1/series/:slug —— 专题详情 + 已发布文章（按专题内序号）
func GetSeries(c *gin.Context) {
	var series model.Series
	if err := model.DB.Where("slug = ? AND visible = ?", c.Param("slug"), true).
		First(&series).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "专题不存在")
		return
	}
	var posts []model.Post
	if err := model.DB.Preload("Category").Preload("Tags").
		Where("series_id = ? AND status = ?", series.ID, model.PostPublished).
		Order("series_sort ASC, id ASC").
		Find(&posts).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	summaries := make([]model.PostSummaryDTO, 0, len(posts))
	for i := range posts {
		summaries = append(summaries, model.ToPostSummary(&posts[i]))
	}
	common.OK(c, gin.H{
		"series": toSeriesDTO(&series, int64(len(summaries))),
		"posts":  summaries,
	})
}
