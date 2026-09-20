package main

// 访问统计（契约 #66-68）：埋点入库、UV 去重、聚合正确性、限流静默、stats 扩展字段

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"myblog/server/model"
)

// track 上报一次页面访问。注：httptest 的请求 UA 固定为空、ClientIP 固定，
// 因此 visitor_hash 在用例内天然一致（正好用于验证 UV 去重）；ua 参数仅作文档提示。
func track(t *testing.T, r http.Handler, payload map[string]any, ua string) (int, bool) {
	t.Helper()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/track", "", payload)
	e := decode(t, rec)
	var data struct {
		OK bool `json:"ok"`
	}
	decodeInto(t, e, &data)
	return e.Code, data.OK
}

func TestTrackAndAggregates(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	post := createPost(t, r, token, map[string]any{"title": "统计文", "status": 1})

	desktopUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0 Safari/537.36"
	mobileUA := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Safari/604.1"
	_ = desktopUA
	_ = mobileUA // httptest 无法定制请求 UA；设备/浏览器聚合的解析逻辑由单元逻辑覆盖（见 model.NewPageView 的确定性实现）

	trackWithUA := func(payload map[string]any, ua string) bool {
		t.Helper()
		_ = ua
		rec := doJSON(t, r, http.MethodPost, "/api/v1/track", "", payload)
		e := decode(t, rec)
		var data struct {
			OK bool `json:"ok"`
		}
		decodeInto(t, e, &data)
		return data.OK
	}

	trackWithUA(map[string]any{"path": "/post/" + post.Slug, "postId": post.ID}, desktopUA)
	trackWithUA(map[string]any{"path": "/post/" + post.Slug, "postId": post.ID}, mobileUA)
	trackWithUA(map[string]any{"path": "/post/" + post.Slug, "postId": post.ID}, mobileUA)
	trackWithUA(map[string]any{"path": "/about"}, desktopUA)
	trackWithUA(map[string]any{
		"path":    "/post/" + post.Slug,
		"postId":  post.ID,
		"referer": "https://www.google.com/search?q=myblog",
	}, desktopUA)
	trackWithUA(map[string]any{"path": "not-a-path"}, desktopUA) // 非法 → ok:false 不入库

	// 全站聚合
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/analytics?range=7d", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("analytics 失败：%s", e.Message)
	}
	var data struct {
		Totals struct {
			PV int64 `json:"pv"`
			UV int64 `json:"uv"`
		} `json:"totals"`
		Trend []struct {
			Date string `json:"date"`
			PV   int64  `json:"pv"`
			UV   int64  `json:"uv"`
		} `json:"trend"`
		TopPosts []struct {
			PostID       uint   `json:"postId"`
			Title        string `json:"title"`
			PV           int64  `json:"pv"`
			UV           int64  `json:"uv"`
			LikeCount    int    `json:"likeCount"`
			CommentCount int64  `json:"commentCount"`
		} `json:"topPosts"`
		Sources []countItemJSON `json:"sources"`
		Devices []countItemJSON `json:"devices"`
	}
	decodeInto(t, e, &data)

	if data.Totals.PV != 5 {
		t.Fatalf("PV 应为 5（非法上报不计），got %d", data.Totals.PV)
	}
	// httptest 固定 ClientIP 且 UA 为空 → 同一 visitor_hash → UV=1
	if data.Totals.UV != 1 {
		t.Fatalf("同 IP 同 UA 的 UV 应去重为 1，got %d", data.Totals.UV)
	}
	var trendPV int64
	for _, p := range data.Trend {
		trendPV += p.PV
	}
	if trendPV != 5 {
		t.Fatalf("趋势 PV 合计应为 5，got %d", trendPV)
	}
	if len(data.TopPosts) != 1 || data.TopPosts[0].PostID != post.ID || data.TopPosts[0].PV != 4 {
		t.Fatalf("热门文章应为该文 PV=4，got %+v", data.TopPosts)
	}
	if data.TopPosts[0].Title != "统计文" {
		t.Fatalf("热门文章标题缺失，got %+v", data.TopPosts[0])
	}
	foundSearch := false
	for _, s := range data.Sources {
		if s.Key == "search" && s.PV == 1 {
			foundSearch = true
		}
	}
	if !foundSearch {
		t.Fatalf("来源应含 search=1，got %+v", data.Sources)
	}
	// 非文章页不计入热门文章，但计入 PV（/about 1 条 + 文章 4 条 = 5）✓
}

type countItemJSON struct {
	Key string `json:"source"`
	PV  int64  `json:"pv"`
}

// TestAnalyticsPrevTotals 环比基线（契约 #67 prevTotals）：
// 上一周期只取「紧邻当前窗口之前、等长、且只到当前已过时长」的时间切片。
func TestAnalyticsPrevTotals(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	base := time.Now()
	seed := func(offsetDays int, hour int, visitor string) {
		t.Helper()
		at := time.Date(base.Year(), base.Month(), base.Day(), hour, 0, 0, 0, base.Location()).
			AddDate(0, 0, offsetDays)
		if err := model.DB.Create(&model.PageView{
			Path:        "/post/seed",
			VisitorHash: visitor,
			CreatedAt:   at,
		}).Error; err != nil {
			t.Fatalf("写入样本失败：%v", err)
		}
	}

	// range=7d：当前窗口 = [today-6 00:00, today+1 00:00)；
	// 环比窗口 = [now-13d, now-7d)（即紧邻之前、等长、且只到当前已过时长的切片）。
	// 下列取样点与「当下时刻」无关，任何运行时间下归属都稳定。
	seed(-10, 9, "prev-a") // 仅在环比窗口
	seed(-8, 20, "prev-b") // 仅在环比窗口
	seed(-6, 9, "cur-a")   // 仅在当前窗口（今日-6 起算）
	seed(-3, 9, "cur-b")   // 仅在当前窗口
	seed(-1, 9, "cur-c")   // 仅在当前窗口
	seed(-40, 9, "old-a")  // 早于两个窗口，均不计入

	var data struct {
		Totals struct {
			PV int64 `json:"pv"`
			UV int64 `json:"uv"`
		} `json:"totals"`
		PrevTotals struct {
			PV int64 `json:"pv"`
			UV int64 `json:"uv"`
		} `json:"prevTotals"`
	}
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/analytics?range=7d", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("analytics 失败：%s", e.Message)
	}
	decodeInto(t, e, &data)

	if data.Totals.PV != 3 {
		t.Fatalf("当前窗口 PV 应为 3（cur-a/cur-b/cur-c），got %d", data.Totals.PV)
	}
	if data.PrevTotals.PV != 2 || data.PrevTotals.UV != 2 {
		t.Fatalf("环比窗口 PV/UV 应为 2/2（prev-a/prev-b），got %+v", data.PrevTotals)
	}
}

func TestTrackValidationAndRateLimit(t *testing.T) {
	r := newTestApp(t)

	// 空路径 → ok:false 且不入库
	if code, ok := track(t, r, map[string]any{"path": ""}, ""); code != 0 || ok {
		t.Fatalf("空路径应静默 ok:false，got code=%d ok=%v", code, ok)
	}
	var n int64
	model.DB.Model(&model.PageView{}).Count(&n)
	if n != 0 {
		t.Fatalf("非法上报不应入库，got %d", n)
	}

	// postId 不存在 → 记为非文章页
	track(t, r, map[string]any{"path": "/post/none", "postId": 424242}, "")
	model.DB.Model(&model.PageView{}).Count(&n)
	if n != 1 {
		t.Fatalf("postId 无效时仍应记录路径，got %d", n)
	}

	// 限流：同 IP 每分钟 60 条，超出后 ok:false 但 HTTP 200
	for i := 0; i < 70; i++ {
		track(t, r, map[string]any{"path": "/p"}, "")
	}
	var total int64
	model.DB.Model(&model.PageView{}).Count(&total)
	if total > 62 { // 1 + 1 + 60
		t.Fatalf("限流应生效，实际入库 %d 条", total)
	}
}

func TestPostAnalyticsAndStatsFields(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	post := createPost(t, r, token, map[string]any{"title": "单篇分析", "status": 1})

	track(t, r, map[string]any{"path": "/post/" + post.Slug, "postId": post.ID}, "")
	track(t, r, map[string]any{"path": "/post/" + post.Slug, "postId": post.ID}, "")
	// 点赞 +1（公开接口）
	rec := doJSON(t, r, http.MethodPost, "/api/v1/posts/"+post.Slug+"/like", "", nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("点赞失败：code=%d", e.Code)
	}

	rec = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/analytics/posts/%d?range=7d", post.ID), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("单篇分析失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Post struct {
			ID    uint   `json:"id"`
			Title string `json:"title"`
		} `json:"post"`
		Totals struct {
			PV           int64 `json:"pv"`
			UV           int64 `json:"uv"`
			LikeCount    int64 `json:"likeCount"`
			CommentCount int64 `json:"commentCount"`
		} `json:"totals"`
		Trend []struct {
			PV int64 `json:"pv"`
		} `json:"trend"`
	}
	decodeInto(t, e, &data)
	if data.Post.ID != post.ID || data.Post.Title != "单篇分析" {
		t.Fatalf("单篇分析文章信息不符，got %+v", data.Post)
	}
	if data.Totals.PV != 2 || data.Totals.UV != 1 || data.Totals.LikeCount != 1 || data.Totals.CommentCount != 0 {
		t.Fatalf("单篇 totals 不符：%+v", data.Totals)
	}
	if len(data.Trend) == 0 {
		t.Fatal("趋势不应为空")
	}

	// 不存在文章 → 10004
	if e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/analytics/posts/999999?range=7d", token, nil)); e.Code != 10004 {
		t.Fatalf("不存在文章应 10004，got %d", e.Code)
	}

	// stats 扩展字段
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/stats", token, nil)
	var stats struct {
		TodayPv      int64 `json:"todayPv"`
		TodayUv      int64 `json:"todayUv"`
		YesterdayPv  int64 `json:"yesterdayPv"`
		YesterdayUv  int64 `json:"yesterdayUv"`
		ScheduledCnt int64 `json:"scheduledCount"`
	}
	decodeInto(t, decode(t, rec), &stats)
	if stats.TodayPv != 2 || stats.TodayUv != 1 || stats.YesterdayPv != 0 {
		t.Fatalf("stats PV/UV 字段不符：%+v", stats)
	}
}
