package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestSitemap(t *testing.T) {
	app := newTestApp(t)

	req := mustNewRequest(t, http.MethodGet, "/sitemap.xml", "")
	rec := httptestRecord(t, app, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("sitemap 应 200，实际 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "xml") {
		t.Fatalf("Content-Type 应为 xml，实际 %s", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`,
		"/posts/gin-blog-backend", // 已发布文章
		"/pages/about",            // 已发布页面
		"/category/uncategorized",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("sitemap 缺少 %q\n%s", want, body[:500])
		}
	}
	// 首页 + 3 篇已发布（草稿不出现）+ 1 页面 + 1 分类 + 4 标签 = 10
	if got := strings.Count(body, "<loc>"); got != 10 {
		t.Fatalf("sitemap loc 应为 10，实际 %d\n%s", got, body)
	}
}

func TestSitemapWithoutSiteURL(t *testing.T) {
	app := newTestApp(t)

	// siteUrl 未设置时回退到请求 Host，链接仍为绝对地址
	req := mustNewRequest(t, http.MethodGet, "/sitemap.xml", "")
	req.Host = "blog.example.com"
	rec := httptestRecord(t, app, req)
	if !strings.Contains(rec.Body.String(), "http://blog.example.com/posts/") {
		t.Fatalf("siteUrl 为空应回退请求 Host\n%s", rec.Body.String()[:300])
	}
}

func TestRobots(t *testing.T) {
	app := newTestApp(t)

	req := mustNewRequest(t, http.MethodGet, "/robots.txt", "")
	rec := httptestRecord(t, app, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("robots 应 200，实际 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Fatalf("Content-Type 应为 text/plain，实际 %s", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"User-agent: *", "Allow: /", "Sitemap: "} {
		if !strings.Contains(body, want) {
			t.Fatalf("robots 缺少 %q\n%s", want, body)
		}
	}
}
