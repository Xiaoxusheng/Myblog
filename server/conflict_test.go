package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"myblog/server/common"
	"myblog/server/model"
)

// adminPutPost PUT 更新文章，返回业务码与响应 envelope（冲突用例需要非 0 码不 fatal）
func adminPutPost(t *testing.T, r http.Handler, token string, id uint, payload map[string]any) envelope {
	t.Helper()
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/posts/%d", id), token, payload)
	return decode(t, rec)
}

// adminPostByID GET 单篇文章
func adminPostByID(t *testing.T, r http.Handler, token string, id uint) model.AdminPostItemDTO {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/posts/%d", id), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("获取文章失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Post model.AdminPostItemDTO `json:"post"`
	}
	decodeInto(t, e, &data)
	return data.Post
}

func fullPostPayload(title, content string, extra map[string]any) map[string]any {
	payload := map[string]any{
		"title":   title,
		"slug":    "",
		"summary": "摘要",
		"content": content,
		"cover":   "",
		"status":  0,
		"isTop":   false,
		"tags":    []string{},
	}
	for k, v := range extra {
		payload[k] = v
	}
	return payload
}

// 模块五并发编辑保护：过期 baseUpdatedAt → 10005 且内容不被覆盖；匹配/缺省 → 正常保存
func TestAdminUpdateConflictGuard(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	post := createPost(t, app, token, map[string]any{"title": "冲突保护", "status": 0, "content": "第一版内容"})
	base := post.UpdatedAt.Format(time.RFC3339Nano)

	// 1) 基线匹配 → 保存成功
	e := adminPutPost(t, app, token, post.ID, fullPostPayload("冲突保护", "第二版内容", map[string]any{"baseUpdatedAt": base}))
	if e.Code != 0 {
		t.Fatalf("基线匹配应保存成功：code=%d msg=%s", e.Code, e.Message)
	}
	// 服务器把 updated_at 截断到毫秒（与浏览器 Date 精度对齐）。两次保存若落在
	// 服务器把 updated_at 截断到毫秒（与浏览器 Date 精度对齐），因此「同一毫秒内
	// 的两次保存」在语义上视为同一版本——这是刻意设计：自动保存连击不该被误报
	// 冲突。本用例要验证的是「基线确实过期」的场景，故直接构造一个明确的旧基线，
	// 不依赖机器速度去赌毫秒边界。
	stale := post.UpdatedAt.Add(-time.Second).Format(time.RFC3339Nano)

	// 2) 过期基线（比当前 updated_at 早 1 秒）→ 10005，内容不被覆盖
	e = adminPutPost(t, app, token, post.ID, fullPostPayload("冲突保护", "过期窗口写入", map[string]any{"baseUpdatedAt": stale}))
	if e.Code != common.CodeConflict {
		t.Fatalf("过期基线应返回 10005，实际 code=%d msg=%s", e.Code, e.Message)
	}
	if current := adminPostByID(t, app, token, post.ID); current.Content != "第二版内容" {
		t.Fatalf("冲突请求不得覆盖内容，实际：%q", current.Content)
	}

	// 2b) 未来基线（比当前晚 1 秒）同样视为不匹配 → 10005（防止客户端时钟超前绕过）
	e = adminPutPost(t, app, token, post.ID, fullPostPayload("冲突保护", "未来窗口写入",
		map[string]any{"baseUpdatedAt": post.UpdatedAt.Add(time.Second).Format(time.RFC3339Nano)}))
	if e.Code != common.CodeConflict {
		t.Fatalf("未来基线应返回 10005，实际 code=%d msg=%s", e.Code, e.Message)
	}

	// 2c) 用「当前服务器值」作为基线 → 应放行（正常保存路径）
	fresh := adminPostByID(t, app, token, post.ID).UpdatedAt.Format(time.RFC3339Nano)
	e = adminPutPost(t, app, token, post.ID, fullPostPayload("冲突保护", "同毫秒写入", map[string]any{"baseUpdatedAt": fresh}))
	if e.Code != 0 {
		t.Fatalf("当前基线应放行，实际 code=%d msg=%s", e.Code, e.Message)
	}

	// 3) 缺省 baseUpdatedAt → 向后兼容，正常保存
	e = adminPutPost(t, app, token, post.ID, fullPostPayload("冲突保护", "第三版内容", nil))
	if e.Code != 0 {
		t.Fatalf("缺省基线应保存成功：code=%d msg=%s", e.Code, e.Message)
	}
	if got := adminPostByID(t, app, token, post.ID); got.Content != "第三版内容" {
		t.Fatalf("缺省基线保存未生效：%q", got.Content)
	}
}

// 模块五草稿工作区：tagId 过滤 + sort=updatedAt 排序
func TestAdminListTagFilterAndUpdatedAtSort(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	postA := createPost(t, app, token, map[string]any{"title": "标签过滤A", "status": 0, "tags": []string{"过滤器甲"}})
	postB := createPost(t, app, token, map[string]any{"title": "标签过滤B", "status": 0, "tags": []string{"过滤器乙"}})

	// 从 admin 列表拿到 tag id
	rec := doJSON(t, app, http.MethodGet, "/api/v1/admin/posts?keyword=标签过滤", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("admin 列表失败：%s", e.Message)
	}
	var list struct {
		List []model.AdminPostItemDTO `json:"list"`
	}
	decodeInto(t, e, &list)
	var tagAID uint
	for _, p := range list.List {
		if p.ID == postA.ID {
			for _, tg := range p.Tags {
				if tg.Name == "过滤器甲" {
					tagAID = tg.ID
				}
			}
		}
	}
	if tagAID == 0 {
		t.Fatal("未取到标签 id")
	}

	// tagId 过滤：仅返回带该标签的文章
	rec = doJSON(t, app, http.MethodGet, fmt.Sprintf("/api/v1/admin/posts?tagId=%d&pageSize=50", tagAID), token, nil)
	e = decode(t, rec)
	decodeInto(t, e, &list)
	if len(list.List) != 1 || list.List[0].ID != postA.ID {
		t.Fatalf("tagId 过滤应仅含文章A，实际 %+v", list.List)
	}

	// sort=updatedAt：更新文章A使其 updated_at 最新 → A 排最前；缺省按创建时间 B 更新于 A 之后，B 应在前
	if e := adminPutPost(t, app, token, postA.ID, fullPostPayload("标签过滤A-renamed", "刷新更新时间", nil)); e.Code != 0 {
		t.Fatalf("更新A失败：%s", e.Message)
	}
	rec = doJSON(t, app, http.MethodGet, "/api/v1/admin/posts?sort=updatedAt&pageSize=50", token, nil)
	e = decode(t, rec)
	decodeInto(t, e, &list)
	if len(list.List) < 2 || list.List[0].ID != postA.ID {
		t.Fatalf("sort=updatedAt 应使刚更新的A居首，实际前2：[%d %d]",
			list.List[0].ID, list.List[1].ID)
	}
	// B 的 updated_at 介于 A 创建与 A 更新之间，应紧随其后（种子文章更早）
	if list.List[1].ID != postB.ID {
		t.Fatalf("sort=updatedAt 第二位应为B（创建晚于种子），实际 %d", list.List[1].ID)
	}
}
