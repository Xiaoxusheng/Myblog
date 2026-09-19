package main

// 搜索统计（#85）与文章 SEO 字段测试

import (
	"fmt"
	"net/http"
	"testing"

	"myblog/server/model"
)

func TestSearchStats(t *testing.T) {
	r := newTestApp(t)
	createPost(t, r, tokenOf(t, r), map[string]any{"title": "Go 教程", "status": 1})

	// 三次搜索：2 次 Go（有结果）、1 次必无结果的词
	publicSearch(t, r, "?keyword=Go")
	publicSearch(t, r, "?keyword=go")
	publicSearch(t, r, "?keyword=zzzznotfound")
	publicSearch(t, r, "") // 空关键词不记录

	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/analytics/searches?range=30d", tokenOf(t, r), nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("搜索统计失败：%s", e.Message)
	}
	var data struct {
		List []struct {
			Keyword       string `json:"keyword"`
			Count         int64  `json:"count"`
			NoResultCount int64  `json:"noResultCount"`
		} `json:"list"`
	}
	decodeInto(t, e, &data)
	if len(data.List) != 2 {
		t.Fatalf("应有 2 个关键词（空词不计），got %+v", data.List)
	}
	var goRow, noneRow *struct {
		Keyword       string `json:"keyword"`
		Count         int64  `json:"count"`
		NoResultCount int64  `json:"noResultCount"`
	}
	for i := range data.List {
		switch data.List[i].Keyword {
		case "go":
			goRow = &data.List[i]
		case "zzzznotfound":
			noneRow = &data.List[i]
		}
	}
	if goRow == nil || goRow.Count != 2 || goRow.NoResultCount != 0 {
		t.Fatalf("Go 搜索应为 2 次有结果，got %+v", goRow)
	}
	if noneRow == nil || noneRow.Count != 1 || noneRow.NoResultCount != 1 {
		t.Fatalf("无结果搜索应计数 1，got %+v", noneRow)
	}
}

func publicSearch(t *testing.T, r http.Handler, query string) {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/posts"+query, "", nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("公开搜索失败：code=%d", e.Code)
	}
}

func tokenOf(t *testing.T, r http.Handler) string { return loginToken(t, r) }

func TestPostSEOFields(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	// 创建带 SEO 字段
	post := createPost(t, r, token, map[string]any{
		"title": "SEO 文", "slug": "seo-post", "status": 1,
		"seoTitle": "SEO 标题", "seoDescription": "SEO 描述",
		"canonical": "https://example.com/canon", "ogImage": "/uploads/a.png",
	})
	if post.SeoTitle != "SEO 标题" || post.Canonical != "https://example.com/canon" {
		t.Fatalf("SEO 字段应写入，got %+v", post)
	}

	// 校验：Canonical 非 http → 10001
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/posts", token, map[string]any{
		"title": "坏 Canonical", "content": "内容", "status": 1, "canonical": "javascript:alert(1)",
	})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("非法 canonical 应 10001，got %d", e.Code)
	}

	// 更新 SEO 字段
	updated := updatePostHelper(t, r, token, post.ID, map[string]any{
		"title": "SEO 文", "slug": "seo-post", "status": 1,
		"seoTitle": "新标题", "seoDescription": "", "canonical": "", "ogImage": "",
	})
	if updated.SeoTitle != "新标题" || updated.Canonical != "" {
		t.Fatalf("SEO 更新失败：%+v", updated)
	}

	// 超长校验
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/posts", token, map[string]any{
		"title": "超长", "content": "x", "status": 1, "seoTitle": fmt.Sprintf("%.200s", stringsRepeat("长", 300)),
	})
	_ = rec
}

func stringsRepeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

var _ = model.PostDraft
