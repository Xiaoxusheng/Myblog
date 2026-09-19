package main

// 专题（契约 #53/#54/#56-61）：CRUD、文章归属与排序、公开可见性、详情 series 上下文

import (
	"fmt"
	"net/http"
	"testing"

	"myblog/server/model"
)

func createSeries(t *testing.T, r http.Handler, token string, fields map[string]any) map[string]any {
	t.Helper()
	payload := map[string]any{"name": "", "visible": true}
	for k, v := range fields {
		payload[k] = v
	}
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/series", token, payload)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建专题失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Series map[string]any `json:"series"`
	}
	decodeInto(t, e, &data)
	return data.Series
}

func seriesID(m map[string]any) uint {
	return uint(m["id"].(float64))
}

func publicSeriesDetail(t *testing.T, r http.Handler, slug string) (total int, ids []uint) {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/series/"+slug, "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("公开专题详情失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Posts []struct {
			ID uint `json:"id"`
		} `json:"posts"`
	}
	decodeInto(t, e, &data)
	for _, p := range data.Posts {
		ids = append(ids, p.ID)
	}
	return len(data.Posts), ids
}

func TestSeriesCRUD(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	s := createSeries(t, r, token, map[string]any{"name": "Go 从入门到实战", "description": "系列教程", "sort": 1})
	if s["name"] != "Go 从入门到实战" || s["slug"] == "" || s["visible"] != true {
		t.Fatalf("创建专题返回异常：%+v", s)
	}
	slug := s["slug"].(string)

	// 重名 → 10001
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/series", token,
		map[string]any{"name": "Go 从入门到实战", "visible": true})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("重名专题应 10001，got code=%d", e.Code)
	}

	// 更新：改名、隐藏
	id := seriesID(s)
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/series/%d", id), token,
		map[string]any{"name": "Go 实战", "slug": slug, "visible": false, "sort": 2})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("更新专题失败：code=%d msg=%s", e.Code, e.Message)
	}

	// 隐藏后公开列表不可见、详情 10004
	publicRec := doJSON(t, r, http.MethodGet, "/api/v1/series", "", nil)
	publicEnv := decode(t, publicRec)
	var pub struct {
		List []map[string]any `json:"list"`
	}
	decodeInto(t, publicEnv, &pub)
	for _, item := range pub.List {
		if item["id"].(float64) == float64(id) {
			t.Fatal("隐藏专题不应出现在公开列表")
		}
	}
	if e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/series/"+slug, "", nil)); e.Code != 10004 {
		t.Fatalf("隐藏专题详情应 10004，got code=%d", e.Code)
	}

	// 删除
	rec = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/series/%d", id), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除专题失败：code=%d", e.Code)
	}
}

func TestSeriesPostMembership(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	s := createSeries(t, r, token, map[string]any{"name": "专题A", "slug": "series-a"})
	sid := seriesID(s)

	// 依序加入 3 篇（不传序号 → 自动 1/2/3）
	p1 := createPost(t, r, token, map[string]any{"title": "P1", "seriesId": sid, "status": 1})
	p2 := createPost(t, r, token, map[string]any{"title": "P2", "seriesId": sid, "status": 1})
	p3 := createPost(t, r, token, map[string]any{"title": "P3", "seriesId": sid, "status": 1})

	// P2 显式调到 5；加入 P4 自动 = max+1 = 4
	updatePostHelper(t, r, token, p2.ID, map[string]any{"title": "P2", "seriesId": sid, "seriesSort": 5})
	p4 := createPost(t, r, token, map[string]any{"title": "P4", "seriesId": sid, "status": 1})

	total, ids := publicSeriesDetail(t, r, "series-a")
	if total != 4 {
		t.Fatalf("专题应含 4 篇，got %d", total)
	}
	// 期望顺序：P1(1) P3(3) P2(5) P4(6)（P2 显式 5，P4 自动 max+1=6）
	expect := []uint{p1.ID, p3.ID, p2.ID, p4.ID}
	for i := range expect {
		if ids[i] != expect[i] {
			t.Fatalf("专题顺序不符：expect %v got %v", expect, ids)
		}
	}

	// P1 换到新专题（一文一专题）
	s2 := createSeries(t, r, token, map[string]any{"name": "专题B", "slug": "series-b"})
	updatePostHelper(t, r, token, p1.ID, map[string]any{"title": "P1", "seriesId": seriesID(s2)})
	if _, ids := publicSeriesDetail(t, r, "series-a"); len(ids) != 3 {
		t.Fatalf("P1 移出后专题 A 应剩 3 篇，got %d", len(ids))
	}

	// P3 移出专题（seriesId=0）
	updatePostHelper(t, r, token, p3.ID, map[string]any{"title": "P3", "seriesId": 0})
	if _, ids := publicSeriesDetail(t, r, "series-a"); len(ids) != 2 {
		t.Fatalf("P3 移出后专题 A 应剩 2 篇，got %d", len(ids))
	}

	// 不存在的专题 → 10001
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/posts", token, map[string]any{
		"title": "孤儿", "content": "内容", "status": 1, "seriesId": 999, "tags": []string{},
	})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("不存在专题应 10001，got code=%d", e.Code)
	}

	// 批量排序：A 内两篇 [P4=1, P2=2]
	postsRec := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/series/%d/posts", sid), token, nil)
	postsEnv := decode(t, postsRec)
	var posts struct {
		List []struct {
			ID   uint `json:"id"`
			Sort int  `json:"sort"`
		} `json:"list"`
	}
	decodeInto(t, postsEnv, &posts)
	items := []map[string]any{
		{"postId": p4.ID, "sort": 1},
		{"postId": p2.ID, "sort": 2},
	}
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/series/%d/posts", sid), token,
		map[string]any{"items": items})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("排序失败：code=%d", e.Code)
	}
	if _, ids := publicSeriesDetail(t, r, "series-a"); ids[0] != p4.ID || ids[1] != p2.ID {
		t.Fatalf("排序后顺序应为 P4,P2，got %v", ids)
	}

	// 公开列表 postCount=已发布数（草稿不计）
	createPost(t, r, token, map[string]any{"title": "草稿P", "seriesId": sid, "status": 0})
	pubRec := doJSON(t, r, http.MethodGet, "/api/v1/series", "", nil)
	pubEnv := decode(t, pubRec)
	var pub struct {
		List []map[string]any `json:"list"`
	}
	decodeInto(t, pubEnv, &pub)
	for _, item := range pub.List {
		if item["id"].(float64) == float64(sid) && item["postCount"].(float64) != 2 {
			t.Fatalf("专题 A 公开 postCount 应为 2，got %v", item["postCount"])
		}
	}
}

func TestPostDetailSeriesContext(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	s := createSeries(t, r, token, map[string]any{"name": "上下文专题", "slug": "ctx-series", "visible": true})
	sid := seriesID(s)
	pa := createPost(t, r, token, map[string]any{"title": "A", "seriesId": sid, "status": 1})
	pb := createPost(t, r, token, map[string]any{"title": "B", "seriesId": sid, "status": 1})
	pc := createPost(t, r, token, map[string]any{"title": "C", "seriesId": sid, "status": 1})

	// 中间篇 B：index=2 total=3，prev=A next=C
	rec := doJSON(t, r, http.MethodGet, "/api/v1/posts/"+pb.Slug, "", nil)
	e := decode(t, rec)
	var detail struct {
		Series     map[string]any `json:"series"`
		SeriesPrev *struct {
			Slug string `json:"slug"`
		} `json:"seriesPrev"`
		SeriesNext *struct {
			Slug string `json:"slug"`
		} `json:"seriesNext"`
	}
	decodeInto(t, e, &detail)
	if detail.Series == nil {
		t.Fatal("详情应包含 series 上下文")
	}
	if detail.Series["index"].(float64) != 2 || detail.Series["total"].(float64) != 3 {
		t.Fatalf("series.index/total 应为 2/3，got %+v", detail.Series)
	}
	if detail.SeriesPrev == nil || detail.SeriesPrev.Slug != pa.Slug {
		t.Fatalf("seriesPrev 应为 A，got %+v", detail.SeriesPrev)
	}
	if detail.SeriesNext == nil || detail.SeriesNext.Slug != pc.Slug {
		t.Fatalf("seriesNext 应为 C，got %+v", detail.SeriesNext)
	}

	// 非专题文章：series 为 null
	plain := createPost(t, r, token, map[string]any{"title": "独立文章", "status": 1})
	rec = doJSON(t, r, http.MethodGet, "/api/v1/posts/"+plain.Slug, "", nil)
	decodeInto(t, decode(t, rec), &detail)
	if detail.Series != nil {
		t.Fatalf("独立文章不应有 series，got %+v", detail.Series)
	}

	// 删除专题 → 成员文章 series_id 清零，详情 series 消失
	rec = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/series/%d", sid), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除专题失败：code=%d", e.Code)
	}
	var post model.Post
	if err := model.DB.First(&post, pb.ID).Error; err != nil || post.SeriesID != 0 {
		t.Fatalf("删除专题后成员应清零，err=%v seriesId=%d", err, post.SeriesID)
	}
}
