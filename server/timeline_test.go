package main

// 时间线（契约 #91-94/#99）：CRUD、校验、排序规则、显隐、关联文章

import (
	"fmt"
	"net/http"
	"testing"
)

func createTimelineEvent(t *testing.T, r http.Handler, token string, fields map[string]any) map[string]any {
	t.Helper()
	payload := map[string]any{
		"title":     "节点",
		"content":   "描述",
		"eventDate": "2026-01-15T00:00:00Z",
		"visible":   true,
	}
	for k, v := range fields {
		payload[k] = v
	}
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/timeline", token, payload)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建时间线节点失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Item map[string]any `json:"item"`
	}
	decodeInto(t, e, &data)
	return data.Item
}

func publicTimeline(t *testing.T, r http.Handler) []map[string]any {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/timeline", "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("公开时间线失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		List []map[string]any `json:"list"`
	}
	decodeInto(t, e, &data)
	return data.List
}

func TestTimelineCRUDAndOrder(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	older := createTimelineEvent(t, r, token, map[string]any{"title": "起步", "eventDate": "2025-06-01T00:00:00Z"})
	newer := createTimelineEvent(t, r, token, map[string]any{"title": "重构", "eventDate": "2026-03-01T00:00:00Z"})
	pinned := createTimelineEvent(t, r, token, map[string]any{"title": "置顶里程碑", "eventDate": "2024-01-01T00:00:00Z", "sort": -1})

	// 默认按日期倒序；sort=-1 置顶
	list := publicTimeline(t, r)
	if len(list) != 3 {
		t.Fatalf("公开时间线应有 3 个节点，got %d", len(list))
	}
	if list[0]["title"] != "置顶里程碑" || list[1]["title"] != "重构" || list[2]["title"] != "起步" {
		t.Fatalf("排序不符：expect 置顶里程碑/重构/起步，got %v/%v/%v",
			list[0]["title"], list[1]["title"], list[2]["title"])
	}
	_ = older
	_ = newer
	_ = pinned

	// 校验：空标题 / 坏日期 / 不存在的关联文章
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/timeline", token,
		map[string]any{"title": "  ", "eventDate": "2026-01-01T00:00:00Z", "visible": true})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("空标题应 10001，got code=%d", e.Code)
	}
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/timeline", token,
		map[string]any{"title": "坏日期", "eventDate": "2026/01/01", "visible": true})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("非 RFC3339 日期应 10001，got code=%d", e.Code)
	}
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/timeline", token,
		map[string]any{"title": "坏文章", "eventDate": "2026-01-01T00:00:00Z", "postId": 999, "visible": true})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("不存在的关联文章应 10001，got code=%d", e.Code)
	}

	// 隐藏后公开不可见，管理列表仍可见
	newerID := uint(newer["id"].(float64))
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/timeline/%d", newerID), token,
		map[string]any{"title": "重构", "eventDate": "2026-03-01T00:00:00Z", "visible": false})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("更新节点失败：code=%d msg=%s", e.Code, e.Message)
	}
	if list := publicTimeline(t, r); len(list) != 2 {
		t.Fatalf("隐藏后公开应为 2 个节点，got %d", len(list))
	}
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/timeline", token, nil)
	var admin struct {
		Total int64 `json:"total"`
	}
	decodeInto(t, decode(t, rec), &admin)
	if admin.Total != 3 {
		t.Fatalf("管理列表应含 3 个节点（含隐藏），got %d", admin.Total)
	}

	// 删除
	rec = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/timeline/%d", newerID), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除节点失败：code=%d", e.Code)
	}
}

func TestTimelinePostAssociation(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	published := createPost(t, r, token, map[string]any{"title": "已发布文章", "status": 1})
	draft := createPost(t, r, token, map[string]any{"title": "草稿文章", "status": 0})

	createTimelineEvent(t, r, token, map[string]any{
		"title": "带文章节点", "postId": published.ID, "eventDate": "2026-02-01T00:00:00Z",
	})
	createTimelineEvent(t, r, token, map[string]any{
		"title": "带草稿节点", "postId": draft.ID, "eventDate": "2026-01-01T00:00:00Z",
	})

	// 公开侧：仅已发布文章携带 post；草稿关联 post=null
	list := publicTimeline(t, r)
	if len(list) != 2 {
		t.Fatalf("公开时间线应有 2 个节点，got %d", len(list))
	}
	first, second := list[0], list[1]
	if first["post"] == nil {
		t.Fatal("已发布文章关联应为 post 对象")
	}
	if second["post"] != nil {
		t.Fatalf("草稿文章不应出现在公开侧，got %+v", second["post"])
	}

	// 管理侧：草稿关联也可见（任意状态）
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/timeline", token, nil)
	var admin struct {
		List []map[string]any `json:"list"`
	}
	decodeInto(t, decode(t, rec), &admin)
	for _, item := range admin.List {
		if item["title"] == "带草稿节点" && item["post"] == nil {
			t.Fatal("管理侧应能看到草稿关联文章")
		}
	}
}
