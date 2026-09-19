package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"myblog/server/model"
)

// RSS：根路径返回 XML 2.0，含种子文章
func TestRSS(t *testing.T) {
	app := newTestApp(t)

	req := mustNewRequest(t, http.MethodGet, "/rss", "")
	rec := httptestRecord(t, app, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("RSS 应 200，实际 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "xml") {
		t.Fatalf("Content-Type 应为 xml，实际 %s", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{`<rss version="2.0"`, "<item>", "<title>", "gin-blog-backend"} {
		if !strings.Contains(body, want) {
			t.Fatalf("RSS 缺少 %q\n%s", want, body[:500])
		}
	}
	// 种子 3 篇已发布 → 3 个 item，草稿不出现
	if got := strings.Count(body, "<item>"); got != 3 {
		t.Fatalf("RSS item 应为 3，实际 %d", got)
	}
	if strings.Contains(body, "blog-redesign-draft") {
		t.Fatal("草稿不应出现在 RSS")
	}
}

// site 端点：Settings 默认值 + 分类/标签含 postCount
func TestSiteEndpoint(t *testing.T) {
	app := newTestApp(t)

	rec := doJSON(t, app, http.MethodGet, "/api/v1/site", "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("site 失败：%s", e.Message)
	}
	var data struct {
		Settings struct {
			SiteName       string `json:"siteName"`
			CommentEnabled bool   `json:"commentEnabled"`
			PostPageSize   int    `json:"postPageSize"`
		} `json:"settings"`
		Categories []model.Category `json:"categories"`
		Tags       []model.Tag      `json:"tags"`
	}
	decodeInto(t, e, &data)

	if data.Settings.SiteName != "My Blog" || !data.Settings.CommentEnabled || data.Settings.PostPageSize != 10 {
		t.Fatalf("Settings 默认值错误：%+v", data.Settings)
	}
	var uncategorized *model.Category
	for i := range data.Categories {
		if data.Categories[i].Name == "未分类" {
			uncategorized = &data.Categories[i]
		}
	}
	if uncategorized == nil || uncategorized.PostCount != 3 {
		t.Fatalf("未分类 postCount 应为 3：%+v", uncategorized)
	}
	var goTag *model.Tag
	for i := range data.Tags {
		if data.Tags[i].Name == "Go" {
			goTag = &data.Tags[i]
		}
	}
	if goTag == nil || goTag.PostCount != 2 { // 种子文章一和三
		t.Fatalf("Go 标签 postCount 应为 2：%+v", goTag)
	}
}

// 归档：按年分组倒序
func TestArchive(t *testing.T) {
	app := newTestApp(t)

	rec := doJSON(t, app, http.MethodGet, "/api/v1/archive", "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("archive 失败：%s", e.Message)
	}
	var years []struct {
		Year  int `json:"year"`
		Items []struct {
			ID    uint   `json:"id"`
			Title string `json:"title"`
			Slug  string `json:"slug"`
		} `json:"items"`
	}
	decodeInto(t, e, &years)
	if len(years) == 0 {
		t.Fatal("归档不应为空")
	}
	if len(years) > 1 && years[0].Year <= years[1].Year {
		t.Fatalf("年份应倒序：%v", years)
	}
	total := 0
	for _, y := range years {
		total += len(y.Items)
	}
	if total != 3 {
		t.Fatalf("归档应为 3 篇已发布，实际 %d", total)
	}
}

// 自定义页面：about 公开可见；草稿/不存在 10004
func TestPublicPages(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	rec := doJSON(t, app, http.MethodGet, "/api/v1/pages/about", "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("about 页面失败：%s", e.Message)
	}
	var data struct {
		Page model.Page `json:"page"`
	}
	decodeInto(t, e, &data)
	if data.Page.Slug != "about" || data.Page.Status != 1 || data.Page.Content == "" {
		t.Fatalf("about 页面错误：%+v", data.Page)
	}

	// 创建草稿页面 → 公开不可见
	rec = doJSON(t, app, http.MethodPost, "/api/v1/admin/pages", token,
		map[string]any{"title": "测试页", "slug": "test-page", "content": "内容", "status": 0})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("创建页面失败：%s", e.Message)
	}
	rec = doJSON(t, app, http.MethodGet, "/api/v1/pages/test-page", "", nil)
	if e := decode(t, rec); e.Code != 10004 {
		t.Fatalf("草稿页面应 10004，实际 %d", e.Code)
	}
	// 不存在的页面
	rec = doJSON(t, app, http.MethodGet, "/api/v1/pages/no-such-page", "", nil)
	if e := decode(t, rec); e.Code != 10004 {
		t.Fatalf("不存在页面应 10004，实际 %d", e.Code)
	}
}

// 友链公开列表：仅 visible、按 sort 升序
func TestPublicLinks(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	// 隐藏一条种子友链（sort 0 的 Go 官网）
	rec := doJSON(t, app, http.MethodGet, "/api/v1/admin/links", token, nil)
	e := decode(t, rec)
	var list struct {
		List []model.Link `json:"list"`
	}
	decodeInto(t, e, &list)
	if len(list.List) != 3 {
		t.Fatalf("种子友链应 3 条，实际 %d", len(list.List))
	}
	first := list.List[0]
	rec = doJSON(t, app, http.MethodPut, "/api/v1/admin/links/"+strconv.FormatUint(uint64(first.ID), 10), token,
		map[string]any{"name": first.Name, "url": first.URL, "logo": first.Logo,
			"description": first.Description, "visible": false, "sort": first.Sort})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("隐藏友链失败：%s", e.Message)
	}

	// 公开列表只剩 2 条
	rec = doJSON(t, app, http.MethodGet, "/api/v1/links", "", nil)
	e = decode(t, rec)
	var pub struct {
		List []model.Link `json:"list"`
	}
	decodeInto(t, e, &pub)
	if len(pub.List) != 2 {
		t.Fatalf("公开友链应 2 条，实际 %d", len(pub.List))
	}
	for _, l := range pub.List {
		if !l.Visible {
			t.Fatal("公开列表不应包含隐藏友链")
		}
	}
	if len(pub.List) >= 2 && pub.List[0].Sort > pub.List[1].Sort {
		t.Fatalf("友链应按 sort 升序：%d > %d", pub.List[0].Sort, pub.List[1].Sort)
	}
}

// 系统设置回写：GET 默认值 → PUT → GET 生效
func TestSettingsRoundtrip(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	settings := adminSettings(t, app, token)
	if settings["siteName"] != "My Blog" {
		t.Fatalf("默认 siteName 错误：%v", settings["siteName"])
	}
	settings["siteName"] = "我的博客"
	settings["siteUrl"] = "https://blog.example.com"
	settings["postPageSize"] = float64(20)
	putSettings(t, app, token, settings)

	after := adminSettings(t, app, token)
	if after["siteName"] != "我的博客" || after["siteUrl"] != "https://blog.example.com" {
		t.Fatalf("设置未生效：%v", after)
	}
}

// ---------- 小助手 ----------

func mustNewRequest(t *testing.T, method, path, token string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatalf("构造请求失败：%v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func httptestRecord(t *testing.T, r http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}
