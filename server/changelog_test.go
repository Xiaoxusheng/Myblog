package main

// 版本发布记录（契约 #95-98/#100）：SemVer 校验、唯一性、草稿/发布流转、删除保护、排序

import (
	"fmt"
	"net/http"
	"testing"
)

func createChangelog(t *testing.T, r http.Handler, token string, fields map[string]any) (map[string]any, int) {
	t.Helper()
	payload := map[string]any{
		"version":    "1.0.0",
		"title":      "首个版本",
		"content":    "## 更新\n- 初始发布",
		"releasedAt": "2026-01-01T00:00:00Z",
		"status":     0,
	}
	for k, v := range fields {
		payload[k] = v
	}
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/changelogs", token, payload)
	e := decode(t, rec)
	if e.Code != 0 {
		return nil, e.Code
	}
	var data struct {
		Item map[string]any `json:"item"`
	}
	decodeInto(t, e, &data)
	return data.Item, 0
}

func publicChangelog(t *testing.T, r http.Handler) []map[string]any {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/changelog", "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("公开版本记录失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		List []map[string]any `json:"list"`
	}
	decodeInto(t, e, &data)
	return data.List
}

func TestChangelogLifecycle(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	// 草稿默认公开不可见
	item, code := createChangelog(t, r, token, nil)
	if code != 0 {
		t.Fatalf("创建版本记录失败：code=%d", code)
	}
	if list := publicChangelog(t, r); len(list) != 0 {
		t.Fatalf("草稿不应出现在公开侧，got %d 条", len(list))
	}

	// 非法版本号
	for _, bad := range []string{"1.2", "a.b.c", "1.2.3.4", "版本1", "1.2.x"} {
		if _, code := createChangelog(t, r, token, map[string]any{"version": bad}); code != 10001 {
			t.Fatalf("非法版本号 %q 应 10001，got code=%d", bad, code)
		}
	}

	// v 前缀归一 + 重复版本号（含 v 前缀形态）→ 10001
	if _, code := createChangelog(t, r, token, map[string]any{"version": "v1.0.0"}); code != 10001 {
		t.Fatalf("重复版本号（v 前缀）应 10001，got code=%d", code)
	}

	// 发布流转：status=1 → 公开可见
	id := uint(item["id"].(float64))
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/changelogs/%d", id), token,
		map[string]any{"version": "1.0.0", "title": "首个版本", "releasedAt": "2026-01-01T00:00:00Z", "status": 1})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("更新版本记录失败：code=%d msg=%s", e.Code, e.Message)
	}
	list := publicChangelog(t, r)
	if len(list) != 1 || list[0]["version"] != "1.0.0" {
		t.Fatalf("发布后公开应可见 1.0.0，got %+v", list)
	}

	// 版本倒序：更晚发布的 v1.10.0 排在 v1.2.0 之前（releasedAt 倒序）
	if _, code := createChangelog(t, r, token, map[string]any{
		"version": "v1.2.0", "releasedAt": "2026-02-01T00:00:00Z", "status": 1,
	}); code != 0 {
		t.Fatalf("创建 v1.2.0 失败：code=%d", code)
	}
	if _, code := createChangelog(t, r, token, map[string]any{
		"version": "1.10.0", "releasedAt": "2026-03-01T00:00:00Z", "status": 1,
	}); code != 0 {
		t.Fatalf("创建 1.10.0 失败：code=%d", code)
	}
	list = publicChangelog(t, r)
	if len(list) != 3 {
		t.Fatalf("公开应为 3 条，got %d", len(list))
	}
	if list[0]["version"] != "1.10.0" || list[1]["version"] != "1.2.0" || list[2]["version"] != "1.0.0" {
		t.Fatalf("版本倒序不符：got %v/%v/%v", list[0]["version"], list[1]["version"], list[2]["version"])
	}

	// 删除保护：已发布不带 force → 10001；带 force → 成功
	rec = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/changelogs/%d", id), token, nil)
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("已发布记录无 force 应 10001，got code=%d", e.Code)
	}
	rec = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/changelogs/%d?force=true", id), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("带 force 删除应成功，code=%d msg=%s", e.Code, e.Message)
	}
}
