package handler

// 管理端访问分析（契约 #67/#68）。
// 聚合策略：范围内明细行（仅取聚合所需列）拉到 Go 侧分桶/去重/分组，
// 规避 SQLite/MySQL 日期函数差异；博客规模数据量下开销可接受，必要索引见 database.md。

import (
	"sort"
	"time"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// parseAnalyticsRange 解析 range 参数为 [start, end)；today=当日 0 点起
func parseAnalyticsRange(c *gin.Context, allowToday bool) (start, end time.Time, name string, ok bool) {
	name = c.Query("range")
	if name == "" {
		name = "7d"
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch name {
	case "today":
		if !allowToday {
			common.Fail(c, common.CodeParamError, "range 仅支持 7d / 30d / 90d")
			return time.Time{}, time.Time{}, name, false
		}
		return today, today.AddDate(0, 0, 1), name, true
	case "7d":
		return today.AddDate(0, 0, -6), today.AddDate(0, 0, 1), name, true
	case "30d":
		return today.AddDate(0, 0, -29), today.AddDate(0, 0, 1), name, true
	case "90d":
		return today.AddDate(0, 0, -89), today.AddDate(0, 0, 1), name, true
	default:
		common.Fail(c, common.CodeParamError, "range 取值不合法")
		return time.Time{}, time.Time{}, name, false
	}
}

type pvRow struct {
	CreatedAt     time.Time `gorm:"column:created_at"`
	VisitorHash   string    `gorm:"column:visitor_hash"`
	PostID        uint      `gorm:"column:post_id"`
	RefererSource string    `gorm:"column:referer_source"`
	DeviceType    string    `gorm:"column:device_type"`
	Browser       string    `gorm:"column:browser"`
	OS            string    `gorm:"column:os"`
}

type countItem struct {
	Key string `json:"source"`
	PV  int64  `json:"pv"`
}

type trendPointPvUv struct {
	Date string `json:"date"`
	PV   int64  `json:"pv"`
	UV   int64  `json:"uv"`
}

// fetchPvRows 拉取范围明细（按需扩展过滤条件）
func fetchPvRows(start, end time.Time, postID uint) []pvRow {
	query := model.DB.Model(&model.PageView{}).
		Select("created_at, visitor_hash, post_id, referer_source, device_type, browser, os").
		Where("created_at >= ? AND created_at < ?", start, end)
	if postID > 0 {
		query = query.Where("post_id = ?", postID)
	}
	var rows []pvRow
	_ = query.Find(&rows).Error // 查询失败按空数据处理，不打断看板
	return rows
}

// aggregateTrend 按日分桶：PV 计数、UV 按 visitor_hash 去重
func aggregateTrend(rows []pvRow, start time.Time, days int) []trendPointPvUv {
	type dayAgg struct {
		pv    int64
		posts map[string]bool
	}
	buckets := make(map[string]*dayAgg, days+1)
	order := make([]string, 0, days+1)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		order = append(order, date)
		buckets[date] = &dayAgg{posts: map[string]bool{}} // 空日期也要有桶
	}
	for _, r := range rows {
		date := r.CreatedAt.Format("2006-01-02")
		agg, ok := buckets[date]
		if !ok {
			agg = &dayAgg{posts: map[string]bool{}}
			buckets[date] = agg
			order = append(order, date) // today 场景补桶
		}
		agg.pv++
		agg.posts[r.VisitorHash] = true
	}
	sort.Strings(order)
	out := make([]trendPointPvUv, 0, len(order))
	for _, date := range order {
		agg := buckets[date]
		if agg == nil {
			out = append(out, trendPointPvUv{Date: date})
			continue
		}
		out = append(out, trendPointPvUv{Date: date, PV: agg.pv, UV: int64(len(agg.posts))})
	}
	return out
}

// aggregateCount 通用分组计数，pv 降序
func aggregateCount(rows []pvRow, keyOf func(pvRow) string) []countItem {
	counts := map[string]int64{}
	for _, r := range rows {
		counts[keyOf(r)]++
	}
	out := make([]countItem, 0, len(counts))
	for k, v := range counts {
		out = append(out, countItem{Key: k, PV: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PV > out[j].PV })
	return out
}

// Analytics GET /api/v1/admin/analytics?range=today|7d|30d|90d
func Analytics(c *gin.Context) {
	start, end, name, ok := parseAnalyticsRange(c, true)
	if !ok {
		return
	}
	rows := fetchPvRows(start, end, 0)

	totals := gin.H{"pv": len(rows), "uv": distinctVisitors(rows)}
	trendDays := map[string]int{"today": 1, "7d": 7, "30d": 30, "90d": 90}[name]
	trend := aggregateTrend(rows, start, trendDays)
	topPosts := topPostRows(rows)
	sources := aggregateCount(rows, func(r pvRow) string { return r.RefererSource })
	devices := aggregateCount(rows, func(r pvRow) string { return r.DeviceType })
	browsers := aggregateCount(rows, func(r pvRow) string { return r.Browser })
	oses := aggregateCount(rows, func(r pvRow) string { return r.OS })

	common.OK(c, gin.H{
		"range":    name,
		"totals":   totals,
		"trend":    trend,
		"topPosts": topPosts,
		"sources":  sources,
		"devices":  devices,
		"browsers": browsers,
		"oses":     oses,
	})
}

// topPostRows 热门文章：pv 降序取 10，附标题/点赞/评论
func topPostRows(rows []pvRow) []gin.H {
	type postAgg struct {
		pv    int64
		posts map[string]bool
	}
	agg := map[uint]*postAgg{}
	for _, r := range rows {
		if r.PostID == 0 {
			continue
		}
		a, ok := agg[r.PostID]
		if !ok {
			a = &postAgg{posts: map[string]bool{}}
			agg[r.PostID] = a
		}
		a.pv++
		a.posts[r.VisitorHash] = true
	}
	ids := make([]uint, 0, len(agg))
	for id := range agg {
		ids = append(ids, id)
	}
	titles := postTitleMap(ids)

	// 点赞数取 posts 表当前值；评论数复用 commentCountsByPost
	var likeRows []struct {
		ID        uint `gorm:"column:id"`
		LikeCount int  `gorm:"column:like_count"`
	}
	if len(ids) > 0 {
		_ = model.DB.Model(&model.Post{}).Select("id, like_count").Where("id IN ?", ids).Scan(&likeRows).Error
	}
	likes := map[uint]int{}
	for _, lr := range likeRows {
		likes[lr.ID] = lr.LikeCount
	}
	commentCounts := commentCountsByPost(ids)

	type postStat struct {
		id uint
		pv int64
		uv int64
	}
	stats := make([]postStat, 0, len(agg))
	for id, a := range agg {
		stats = append(stats, postStat{id: id, pv: a.pv, uv: int64(len(a.posts))})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].pv != stats[j].pv {
			return stats[i].pv > stats[j].pv
		}
		return stats[i].id < stats[j].id
	})
	if len(stats) > 10 {
		stats = stats[:10]
	}
	out := make([]gin.H, 0, len(stats))
	for _, s := range stats {
		out = append(out, gin.H{
			"postId":       s.id,
			"title":        titles[s.id],
			"pv":           s.pv,
			"uv":           s.uv,
			"likeCount":    likes[s.id],
			"commentCount": commentCounts[s.id],
		})
	}
	return out
}

func distinctVisitors(rows []pvRow) int64 {
	set := map[string]bool{}
	for _, r := range rows {
		set[r.VisitorHash] = true
	}
	return int64(len(set))
}

// PostAnalytics GET /api/v1/admin/analytics/posts/:id?range=7d|30d|90d
func PostAnalytics(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "文章 id 不合法")
		return
	}
	var post model.Post
	if err := model.DB.Select("id", "title").First(&post, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}
	start, end, name, ok := parseAnalyticsRange(c, false)
	if !ok {
		return
	}
	rows := fetchPvRows(start, end, id)

	var commentCount, likeCount int64
	model.DB.Model(&model.Comment{}).Where("post_id = ?", post.ID).Count(&commentCount)
	model.DB.Model(&model.Post{}).Where("id = ?", post.ID).Pluck("COALESCE(like_count, 0)", &likeCount)

	common.OK(c, gin.H{
		"post":  gin.H{"id": post.ID, "title": post.Title},
		"range": name,
		"totals": gin.H{
			"pv":           len(rows),
			"uv":           distinctVisitors(rows),
			"likeCount":    likeCount,
			"commentCount": commentCount,
		},
		"trend":   aggregateTrend(rows, start, map[string]int{"7d": 7, "30d": 30, "90d": 90}[name]),
		"sources": aggregateCount(rows, func(r pvRow) string { return r.RefererSource }),
		"devices": aggregateCount(rows, func(r pvRow) string { return r.DeviceType }),
	})
}
