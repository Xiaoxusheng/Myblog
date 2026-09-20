package main

// 版本发布记录补充加固测试（契约 #95-98）：管理筛选边界、sort 置顶排序、更新唯一性排除自身、字段上限、鉴权
// 复用 changelog_test.go 的 createChangelog / publicChangelog 助手

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func adminChangelogFilter(t *testing.T, r http.Handler, token, query string) (int, int) {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/changelogs"+query, token, nil)
	e := decode(t, rec)
	if e.Code != 0 && e.Code != 10001 {
		t.Fatalf("管理列表请求失败：code=%d msg=%s", e.Code, e.Message)
	}
	if e.Code == 10001 {
		return e.Code, 0
	}
	var data struct {
		Total int64 `json:"total"`
	}
	decodeInto(t, e, &data)
	return 0, int(data.Total)
}

func TestChangelogFilterAndFieldLimits(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	if _, code := createChangelog(t, r, token, map[string]any{"version": "0.1.0", "status": 0}); code != 0 {
		t.Fatalf("创建草稿失败：code=%d", code)
	}
	if _, code := createChangelog(t, r, token, map[string]any{"version": "0.2.0", "status": 1}); code != 0 {
		t.Fatalf("创建已发布失败：code=%d", code)
	}

	// 筛选：status=0 → 仅草稿；status=1 → 仅已发布；status=5 → 10001
	if code, n := adminChangelogFilter(t, r, token, "?status=0"); code != 0 || n != 1 {
		t.Fatalf("status=0 应恰 1 条，got code=%d n=%d", code, n)
	}
	if code, n := adminChangelogFilter(t, r, token, "?status=1"); code != 0 || n != 1 {
		t.Fatalf("status=1 应恰 1 条，got code=%d n=%d", code, n)
	}
	if code, _ := adminChangelogFilter(t, r, token, "?status=5"); code != 10001 {
		t.Fatalf("status=5 应 10001，got %d", code)
	}

	// 字段上限：标题 >100 字、内容 >20000 字 → 10001
	if _, code := createChangelog(t, r, token, map[string]any{
		"version": "0.3.0", "title": strings.Repeat("标", 101)}); code != 10001 {
		t.Fatalf("标题超 100 字应 10001，got %d", code)
	}
	if _, code := createChangelog(t, r, token, map[string]any{
		"version": "0.3.0", "content": strings.Repeat("字", 20001)}); code != 10001 {
		t.Fatalf("内容超 20000 字应 10001，got %d", code)
	}
	// releasedAt 非法格式 → 10001
	if _, code := createChangelog(t, r, token, map[string]any{
		"version": "0.3.0", "releasedAt": "2026-09-01"}); code != 10001 {
		t.Fatalf("releasedAt 非 RFC3339 应 10001，got %d", code)
	}
}

func TestChangelogSortPinningAndSelfVersion(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	seed := []map[string]any{
		{"version": "1.0.0", "releasedAt": "2026-01-01T00:00:00Z", "status": 1},
		{"version": "2.0.0", "releasedAt": "2026-06-01T00:00:00Z", "status": 1},
		{"version": "0.9.0", "releasedAt": "2026-03-01T00:00:00Z", "status": 1, "sort": -1},
	}
	for _, s := range seed {
		if _, code := createChangelog(t, r, token, s); code != 0 {
			t.Fatalf("seed %v 失败：code=%d", s, code)
		}
	}
	// sort=-1 置顶在前，其余按发布日期倒序
	pub := publicChangelog(t, r)
	want := []string{"0.9.0", "2.0.0", "1.0.0"}
	for i, w := range want {
		if got := pub[i]["version"]; got != w {
			t.Fatalf("公开排序不符：want %v got %v", want, []any{pub[0]["version"], pub[1]["version"], pub[2]["version"]})
		}
	}

	// 更新唯一性：改成他人已占用版本 → 10001；改回自身（带 v 前缀形态）→ 通过
	var first map[string]any
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/changelogs", token,
		map[string]any{"version": "3.0.0", "releasedAt": "2026-07-01T00:00:00Z", "status": 1})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建 3.0.0 失败：code=%d", e.Code)
	}
	var data struct {
		Item map[string]any `json:"item"`
	}
	decodeInto(t, e, &data)
	first = data.Item
	id := int(first["id"].(float64))

	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/changelogs/%d", id), token,
		map[string]any{"version": "1.0.0", "releasedAt": "2026-07-01T00:00:00Z", "status": 1})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("更新为已占用版本号应 10001，got %d", e.Code)
	}
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/changelogs/%d", id), token,
		map[string]any{"version": "v3.0.0", "releasedAt": "2026-07-01T00:00:00Z", "status": 1})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("保持自身版本号（v 前缀形态）应通过，got code=%d msg=%s", e.Code, e.Message)
	}
}

func TestChangelogUnauthorized(t *testing.T) {
	r := newTestApp(t)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/admin/changelogs"},
		{http.MethodPost, "/api/v1/admin/changelogs"},
		{http.MethodPut, "/api/v1/admin/changelogs/1"},
		{http.MethodDelete, "/api/v1/admin/changelogs/1"},
	} {
		rec := doJSON(t, r, tc.method, tc.path, "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s 未登录应 401，got %d", tc.method, tc.path, rec.Code)
		}
	}
}
