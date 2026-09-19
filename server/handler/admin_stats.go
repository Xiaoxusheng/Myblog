package handler

import (
	"time"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// Stats GET /api/v1/admin/stats —— 计数卡片 + 近 7 天发布/评论趋势 + 最近 5 条评论
func Stats(c *gin.Context) {
	db := model.DB

	var postCount, draftCount, commentCount, pendingCount, linkCount int64
	if err := db.Model(&model.Post{}).Count(&postCount).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	if err := db.Model(&model.Post{}).Where("status = ?", model.PostDraft).Count(&draftCount).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	if err := db.Model(&model.Comment{}).Count(&commentCount).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	if err := db.Model(&model.Comment{}).Where("status = ?", model.CommentPending).Count(&pendingCount).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	if err := db.Model(&model.Link{}).Count(&linkCount).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	var sums struct {
		Views int64 `gorm:"column:views"`
		Likes int64 `gorm:"column:likes"`
	}
	if err := db.Model(&model.Post{}).
		Select("COALESCE(SUM(view_count), 0) AS views, COALESCE(SUM(like_count), 0) AS likes").
		Scan(&sums).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	// 近 7 天趋势：按本地日期分桶（取 created_at 到 Go 侧计算，规避数据库时区差异）
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := today.AddDate(0, 0, -6)
	end := today.AddDate(0, 0, 1)

	var postTimes, commentTimes []time.Time
	if err := db.Model(&model.Post{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Pluck("created_at", &postTimes).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	if err := db.Model(&model.Comment{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Pluck("created_at", &commentTimes).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	postBuckets := make(map[string]int64, 8)
	for _, t := range postTimes {
		postBuckets[t.Format("2006-01-02")]++
	}
	commentBuckets := make(map[string]int64, 8)
	for _, t := range commentTimes {
		commentBuckets[t.Format("2006-01-02")]++
	}

	type trendPoint struct {
		Date     string `json:"date"`
		Posts    int64  `json:"posts"`
		Comments int64  `json:"comments"`
	}
	trend := make([]trendPoint, 0, 7)
	for i := 0; i < 7; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		trend = append(trend, trendPoint{
			Date:     date,
			Posts:    postBuckets[date],
			Comments: commentBuckets[date],
		})
	}

	// 最近 5 条评论（含未审核），附文章标题
	var comments []model.Comment
	if err := db.Order("id DESC").Limit(5).Find(&comments).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	ids := make([]uint, 0, len(comments))
	for i := range comments {
		ids = append(ids, comments[i].PostID)
	}
	titles := postTitleMap(ids)

	type recentComment struct {
		ID        uint      `json:"id"`
		PostTitle string    `json:"postTitle"`
		Nickname  string    `json:"nickname"`
		Content   string    `json:"content"`
		Status    int8      `json:"status"`
		CreatedAt time.Time `json:"createdAt"`
	}
	recent := make([]recentComment, 0, len(comments))
	for i := range comments {
		recent = append(recent, recentComment{
			ID:        comments[i].ID,
			PostTitle: titles[comments[i].PostID],
			Nickname:  comments[i].Nickname,
			Content:   comments[i].Content,
			Status:    comments[i].Status,
			CreatedAt: comments[i].CreatedAt,
		})
	}

	common.OK(c, gin.H{
		"postCount":           postCount,
		"draftCount":          draftCount,
		"commentCount":        commentCount,
		"pendingCommentCount": pendingCount,
		"viewCount":           sums.Views,
		"likeCount":           sums.Likes,
		"linkCount":           linkCount,
		"trend":               trend,
		"recentComments":      recent,
	})
}
