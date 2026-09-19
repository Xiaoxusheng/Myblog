package main

// URL 重定向（契约 #55/#62-65）：CRUD、环检测、resolve、slug 变更自动 301

import (
	"fmt"
	"net/http"
	"testing"
)

type redirectRow struct {
	ID      uint   `json:"id"`
	Source  string `json:"source"`
	Target  string `json:"target"`
	Type    int    `json:"type"`
	Enabled bool   `json:"enabled"`
}

func redirectList(t *testing.T, r http.Handler, token string) []redirectRow {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/redirects?page=1&pageSize=50", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("重定向列表失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		List []redirectRow `json:"list"`
	}
	decodeInto(t, e, &data)
	return data.List
}

func createRedirectRaw(t *testing.T, r http.Handler, token string, payload map[string]any) (int, string) {
	t.Helper()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/redirects", token, payload)
	e := decode(t, rec)
	return e.Code, e.Message
}

func resolveRedirect(t *testing.T, r http.Handler, path string) *redirectRow {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet,
		fmt.Sprintf("/api/v1/redirects/resolve?path=%s", path), "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("resolve 失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Redirect *redirectRow `json:"redirect"`
	}
	decodeInto(t, e, &data)
	return data.Redirect
}

func TestRedirectCRUDAndResolve(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	// 创建：路径自动补前导斜杠
	code, msg := createRedirectRaw(t, r, token, map[string]any{
		"source": "post/a", "target": "/post/b", "type": 301, "enabled": true})
	if code != 0 {
		t.Fatalf("创建重定向失败：code=%d msg=%s", code, msg)
	}
	if got := resolveRedirect(t, r, "/post/a"); got == nil || got.Target != "/post/b" || got.Type != 301 {
		t.Fatalf("resolve 应命中 /post/b 301，got %+v", got)
	}

	// 重复 source → 10001
	if code, _ := createRedirectRaw(t, r, token, map[string]any{
		"source": "/post/a", "target": "/post/c", "type": 301, "enabled": true}); code != 10001 {
		t.Fatalf("重复 source 应 10001，got code=%d", code)
	}

	// source == target、非法 type → 10001
	if code, _ := createRedirectRaw(t, r, token, map[string]any{
		"source": "/post/x", "target": "/post/x", "type": 301, "enabled": true}); code != 10001 {
		t.Fatalf("source==target 应 10001，got %d", code)
	}
	if code, _ := createRedirectRaw(t, r, token, map[string]any{
		"source": "/post/x", "target": "/post/y", "type": 300, "enabled": true}); code != 10001 {
		t.Fatalf("type=300 应 10001，got %d", code)
	}

	// 环检测：/post/b → /post/a 会形成 a→b→a 循环 → 10001
	if code, _ := createRedirectRaw(t, r, token, map[string]any{
		"source": "/post/b", "target": "/post/a", "type": 301, "enabled": true}); code != 10001 {
		t.Fatalf("环重定向应 10001，got code=%d", code)
	}

	// disabled 规则不参与 resolve
	if code, _ := createRedirectRaw(t, r, token, map[string]any{
		"source": "/post/off", "target": "/post/b", "type": 302, "enabled": false}); code != 0 {
		t.Fatalf("创建禁用规则失败：code=%d", code)
	}
	if got := resolveRedirect(t, r, "/post/off"); got != nil {
		t.Fatalf("禁用规则不应被 resolve，got %+v", got)
	}

	// 更新：改 target 后 resolve 反映；无匹配 → null
	list := redirectList(t, r, token)
	var aRule redirectRow
	for _, row := range list {
		if row.Source == "/post/a" {
			aRule = row
		}
	}
	rec := doJSON(t, r, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/redirects/%d", aRule.ID), token,
		map[string]any{"source": "/post/a", "target": "/post/c", "type": 302, "enabled": true})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("更新重定向失败：code=%d", e.Code)
	}
	if got := resolveRedirect(t, r, "/post/a"); got == nil || got.Target != "/post/c" || got.Type != 302 {
		t.Fatalf("更新后 resolve 应命中 /post/c 302，got %+v", got)
	}

	// 删除 → null
	rec = doJSON(t, r, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/redirects/%d", aRule.ID), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除重定向失败：code=%d", e.Code)
	}
	if got := resolveRedirect(t, r, "/post/a"); got != nil {
		t.Fatalf("删除后 resolve 应为 null，got %+v", got)
	}
}

func TestSlugChangeAutoRedirect(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	post := createPost(t, r, token, map[string]any{"title": "迁移文", "slug": "old-slug"})
	updatePostHelper(t, r, token, post.ID, map[string]any{"title": "迁移文", "slug": "new-slug"})

	// 默认开启：自动产生 /post/old-slug → /post/new-slug 301
	got := resolveRedirect(t, r, "/post/old-slug")
	if got == nil || got.Target != "/post/new-slug" || got.Type != 301 {
		t.Fatalf("slug 变更应自动建 301，got %+v", got)
	}

	// 关闭开关后不再自动创建
	rec := doJSON(t, r, http.MethodPut, "/api/v1/admin/settings", token, map[string]any{
		"siteName": "My Blog", "commentEnabled": true, "postPageSize": 10,
		"autoRedirectOnSlugChange": false,
	})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("更新设置失败：code=%d", e.Code)
	}
	before := len(redirectList(t, r, token))
	updatePostHelper(t, r, token, post.ID, map[string]any{"title": "迁移文", "slug": "new-slug-2"})
	if after := len(redirectList(t, r, token)); after != before {
		t.Fatalf("开关关闭不应新增重定向，before=%d after=%d", before, after)
	}
}
