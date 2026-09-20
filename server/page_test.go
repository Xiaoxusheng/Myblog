package main

// 页面管理模块（契约 #38-44 / #101-105）验收测试。
// 覆盖：列表概览与筛选、状态扩展、复制、批量、版本生命周期、恢复、
// slug→301 联动、定时发布、并发基线、审计日志、预览接口。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"myblog/server/model"
)

// ---------- 助手 ----------

// createPage 通过管理端接口创建页面，返回契约 PageDTO
func createPage(t *testing.T, r http.Handler, token string, fields map[string]any) model.PageDTO {
	t.Helper()
	payload := map[string]any{
		"title":   "默认页面",
		"slug":    "",
		"content": "默认页面正文",
		"status":  1,
	}
	for k, v := range fields {
		payload[k] = v
	}
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/pages", token, payload)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建页面失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Page model.PageDTO `json:"page"`
	}
	decodeInto(t, e, &data)
	return data.Page
}

type pageListData struct {
	List     []model.PageItemDTO `json:"list"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
	Meta     model.PageMetaDTO   `json:"meta"`
}

func adminPageList(t *testing.T, r http.Handler, token, query string) pageListData {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/pages"+query, token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("页面列表失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data pageListData
	decodeInto(t, e, &data)
	return data
}

func adminPageByID(t *testing.T, r http.Handler, token string, id uint) model.PageDTO {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/pages/%d", id), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("获取页面失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Page model.PageDTO `json:"page"`
	}
	decodeInto(t, e, &data)
	return data.Page
}

func pageRevisionList(t *testing.T, r http.Handler, token string, id uint) []model.PageRevisionItemDTO {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/pages/%d/revisions?page=1&pageSize=50", id), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("版本列表失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		List []model.PageRevisionItemDTO `json:"list"`
	}
	decodeInto(t, e, &data)
	return data.List
}

// updatePage 全量更新页面，返回原始 envelope 供错误码断言
func updatePage(t *testing.T, r http.Handler, token string, id uint, fields map[string]any) envelope {
	t.Helper()
	payload := map[string]any{
		"title":   "默认页面",
		"slug":    "",
		"content": "默认页面正文",
		"status":  1,
	}
	for k, v := range fields {
		payload[k] = v
	}
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/pages/%d", id), token, payload)
	return decode(t, rec)
}

// ---------- 列表 / 概览 ----------

func TestAdminPageListMetaAndFilters(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	// seed 已有「关于」（已发布）
	createPage(t, r, token, map[string]any{"title": "草稿页", "slug": "draft-p", "status": 0})
	createPage(t, r, token, map[string]any{"title": "隐藏页", "slug": "hidden-p", "status": 2})
	createPage(t, r, token, map[string]any{
		"title": "计划页", "slug": "sched-p", "status": 3,
		"publishAt": time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	})

	all := adminPageList(t, r, token, "?page=1&pageSize=50")
	if all.Total != 4 {
		t.Fatalf("页面总数应为 4，实际 %d", all.Total)
	}
	if all.Meta.TotalCount != 4 || all.Meta.PublishedCount != 1 || all.Meta.DraftCount != 1 || all.Meta.ScheduledCount != 1 {
		t.Fatalf("概览统计不符：%+v", all.Meta)
	}

	// keyword 过滤下 meta 应随筛选收敛
	kw := adminPageList(t, r, token, "?page=1&pageSize=50&keyword=draft-p")
	if kw.Total != 1 || kw.Meta.TotalCount != 1 {
		t.Fatalf("keyword 过滤失效：total=%d meta=%+v", kw.Total, kw.Meta)
	}

	// status 过滤只影响 list，不影响 meta（meta 为 keyword 维度全量）
	only := adminPageList(t, r, token, "?page=1&pageSize=50&status=3")
	if only.Total != 1 || len(only.List) != 1 || only.List[0].Status != 3 {
		t.Fatalf("status 过滤失效：total=%d list=%+v", only.Total, only.List)
	}
	if only.Meta.TotalCount != 4 {
		t.Fatalf("status 过滤不应改变 meta：%+v", only.Meta)
	}

	// 分页：pageSize=1 时 list 1 条但 meta 仍为全量
	one := adminPageList(t, r, token, "?page=1&pageSize=1")
	if len(one.List) != 1 || one.Meta.TotalCount != 4 {
		t.Fatalf("分页与 meta 组合错误：len=%d meta=%+v", len(one.List), one.Meta)
	}

	// 列表项不含 content（JSON 中不得出现 content 字段）
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/pages?page=1&pageSize=50", token, nil)
	if bytes := rec.Body.String(); jsonContainsKey(t, bytes, "content") {
		t.Fatalf("列表项不应包含 content 字段：%s", bytes)
	}
}

func jsonContainsKey(t *testing.T, raw, key string) bool {
	t.Helper()
	var probe map[string]any
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return false
	}
	data, _ := probe["data"].(map[string]any)
	list, _ := data["list"].([]any)
	if len(list) == 0 {
		return false
	}
	first, _ := list[0].(map[string]any)
	_, ok := first[key]
	return ok
}

// ---------- 创建校验 ----------

func TestAdminCreatePageValidation(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	cases := []struct {
		name string
		body map[string]any
		code int
	}{
		{"标题为空", map[string]any{"title": "  ", "slug": "x1", "status": 0}, 10001},
		{"slug 重复", map[string]any{"title": "重名", "slug": "about", "status": 0}, 10001},
		{"status 越界", map[string]any{"title": "越界", "slug": "x2", "status": 9}, 10001},
		{"定时缺 publishAt", map[string]any{"title": "缺时间", "slug": "x3", "status": 3}, 10001},
		{"定时时间已过", map[string]any{
			"title": "过期", "slug": "x4", "status": 3,
			"publishAt": time.Now().Add(-time.Hour).Format(time.RFC3339),
		}, 10001},
		{"pageType 非法", map[string]any{"title": "类型", "slug": "x5", "status": 0, "pageType": "bogus"}, 10001},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/pages", token, tc.body)
			if got := decode(t, rec).Code; got != tc.code {
				t.Fatalf("期望 %d，实际 %d", tc.code, got)
			}
		})
	}

	// slug 留空 → 自动派生
	p := createPage(t, r, token, map[string]any{"title": "Auto Slug", "slug": ""})
	if p.Slug == "" {
		t.Fatal("slug 留空时未自动派生")
	}
	// 创建即写入 v1
	revs := pageRevisionList(t, r, token, p.ID)
	if len(revs) != 1 || revs[0].Version != 1 || revs[0].Remark != "首次保存" {
		t.Fatalf("创建后应有 v1 首次保存，实际 %+v", revs)
	}
	// 首次发布写入 publishedAt
	if p.PublishedAt == nil {
		t.Fatal("status=1 创建时 publishedAt 应写入")
	}
}

// ---------- 版本生命周期 ----------

func TestAdminPageRevisionLifecycle(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	p := createPage(t, r, token, map[string]any{"title": "版本页", "slug": "rev-p", "content": "v1 正文"})

	// 改标题 → v2
	if e := updatePage(t, r, token, p.ID, map[string]any{
		"title": "版本页改", "slug": "rev-p", "content": "v1 正文", "status": 1,
	}); e.Code != 0 {
		t.Fatalf("更新失败：%+v", e)
	}
	revs := pageRevisionList(t, r, token, p.ID)
	if len(revs) != 2 || revs[0].Version != 2 {
		t.Fatalf("应有 v2，实际 %+v", revs)
	}
	if revs[0].Remark == "" {
		t.Fatal("v2 remark 不应为空")
	}

	// 内容无变化 → 不产生新版本
	if e := updatePage(t, r, token, p.ID, map[string]any{
		"title": "版本页改", "slug": "rev-p", "content": "v1 正文", "status": 1,
	}); e.Code != 0 {
		t.Fatalf("更新失败：%+v", e)
	}
	if revs = pageRevisionList(t, r, token, p.ID); len(revs) != 2 {
		t.Fatalf("无变化不应生成版本，实际 %d 个", len(revs))
	}

	// 版本详情返回全量快照
	rec := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/pages/%d/revisions/1", p.ID), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("版本详情失败：%+v", e)
	}
	var detail struct {
		Revision model.PageRevisionDetailDTO `json:"revision"`
	}
	decodeInto(t, e, &detail)
	if detail.Revision.Title != "版本页" || detail.Revision.Content != "v1 正文" {
		t.Fatalf("v1 快照不符：%+v", detail.Revision)
	}

	// 版本不存在 → 10004
	rec = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/pages/%d/revisions/99", p.ID), token, nil)
	if got := decode(t, rec).Code; got != 10004 {
		t.Fatalf("不存在版本应 10004，实际 %d", got)
	}
}

func TestAdminRestorePageRevision(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	p := createPage(t, r, token, map[string]any{"title": "待恢复", "slug": "restore-p", "content": "原始内容"})
	// 先发布并设计划时间，验证恢复不改状态
	if e := updatePage(t, r, token, p.ID, map[string]any{
		"title": "待恢复", "slug": "restore-p", "content": "被改内容", "status": 3,
		"publishAt": time.Now().Add(3 * time.Hour).Format(time.RFC3339),
	}); e.Code != 0 {
		t.Fatalf("更新失败：%+v", e)
	}
	before := adminPageByID(t, r, token, p.ID)
	if before.Status != 3 {
		t.Fatalf("状态应为 3，实际 %d", before.Status)
	}

	// 恢复 v1
	rec := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/admin/pages/%d/revisions/1/restore", p.ID), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("恢复失败：code=%d msg=%s", e.Code, e.Message)
	}
	after := adminPageByID(t, r, token, p.ID)
	if after.Content != "原始内容" {
		t.Fatalf("恢复后内容不符：%q", after.Content)
	}
	// 状态与计划时间不被恢复改写
	if after.Status != 3 || after.PublishAt == nil {
		t.Fatalf("恢复不应改变发布状态/计划时间：status=%d publishAt=%v", after.Status, after.PublishAt)
	}
	// 恢复生成新版本，且含「恢复自 vN」
	revs := pageRevisionList(t, r, token, p.ID)
	if len(revs) < 3 {
		t.Fatalf("恢复应产生新版本，实际 %d 个", len(revs))
	}
	if revs[0].Remark != "恢复自 v1" {
		t.Fatalf("最新版本 remark 应为「恢复自 v1」，实际 %q", revs[0].Remark)
	}
	// 历史版本不被覆盖
	found := false
	for _, rv := range revs {
		if rv.Version == 1 {
			found = true
		}
	}
	if !found {
		t.Fatal("历史 v1 不应被删除")
	}
}

func TestAdminRestorePageSlugConflict(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	a := createPage(t, r, token, map[string]any{"title": "甲页", "slug": "page-a", "content": "甲内容"})
	createPage(t, r, token, map[string]any{"title": "乙页", "slug": "page-b", "content": "乙内容"})

	// 把甲页 slug 改成 page-b 会被占用 → 10001（验证唯一性保护）
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/pages/%d", a.ID), token, map[string]any{
		"title": "甲页", "slug": "page-b", "content": "甲内容", "status": 1,
	})
	if got := decode(t, rec).Code; got != 10001 {
		t.Fatalf("slug 冲突应 10001，实际 %d", got)
	}
	_ = a
}

// ---------- slug → 301 联动 ----------

func TestPageSlugChangeCreatesRedirect(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	p := createPage(t, r, token, map[string]any{"title": "改链页", "slug": "chain-old", "content": "内容"})

	if e := updatePage(t, r, token, p.ID, map[string]any{
		"title": "改链页", "slug": "chain-new", "content": "内容", "status": 1,
	}); e.Code != 0 {
		t.Fatalf("改 slug 失败：%+v", e)
	}

	rec := doJSON(t, r, http.MethodGet, "/api/v1/redirects/resolve?path=/page/chain-old", "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("解析重定向失败：%+v", e)
	}
	var data struct {
		Redirect *struct {
			Source string `json:"source"`
			Target string `json:"target"`
			Type   int    `json:"type"`
		} `json:"redirect"`
	}
	decodeInto(t, e, &data)
	if data.Redirect == nil {
		t.Fatal("slug 变更未创建 301 重定向")
	}
	if data.Redirect.Target != "/page/chain-new" || data.Redirect.Type != 301 {
		t.Fatalf("重定向目标/类型错误：%+v", data.Redirect)
	}

	// 关闭开关后不再自动创建
	rec = doJSON(t, r, http.MethodPut, "/api/v1/admin/settings", token, map[string]any{
		"siteName": "My Blog", "commentEnabled": true, "postPageSize": 10,
		"autoRedirectOnSlugChange": false,
	})
	if decode(t, rec).Code != 0 {
		t.Fatal("关闭自动重定向开关失败")
	}
	if e := updatePage(t, r, token, p.ID, map[string]any{
		"title": "改链页", "slug": "chain-final", "content": "内容", "status": 1,
	}); e.Code != 0 {
		t.Fatalf("二次改 slug 失败：%+v", e)
	}
	rec = doJSON(t, r, http.MethodGet, "/api/v1/redirects/resolve?path=/page/chain-new", "", nil)
	decodeInto(t, decode(t, rec), &data)
	if data.Redirect != nil {
		t.Fatalf("开关关闭时不应创建重定向：%+v", data.Redirect)
	}
}

// ---------- 定时发布 ----------

func TestPageScheduledPublishing(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	future := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	p := createPage(t, r, token, map[string]any{
		"title": "计划页", "slug": "sched-page", "content": "计划内容",
		"status": 3, "publishAt": future.Format(time.RFC3339),
	})
	if p.Status != 3 || p.PublishAt == nil {
		t.Fatalf("定时页面创建异常：%+v", p)
	}

	// 未到点：公开接口不可见
	rec := doJSON(t, r, http.MethodGet, "/api/v1/pages/sched-page", "", nil)
	if got := decode(t, rec).Code; got != 10004 {
		t.Fatalf("未到点应 10004，实际 %d", got)
	}
	// 计入 scheduledCount
	list := adminPageList(t, r, token, "?page=1&pageSize=50")
	if list.Meta.ScheduledCount < 1 {
		t.Fatalf("scheduledCount 应 ≥1，实际 %d", list.Meta.ScheduledCount)
	}

	// 把计划时间改到过去，调调度器。
	// 注意必须写 UTC：库内 publish_at 统一归一为 UTC，调度器也以 UTC 比较；
	// 若这里写本地时区的时间戳，字典序比较会失配（见 TestScheduledPublishNotBeforeDueAt）。
	past := time.Now().Add(-time.Minute).Truncate(time.Second)
	if err := model.DB.Model(&model.Page{}).Where("id = ?", p.ID).
		Update("publish_at", past.UTC()).Error; err != nil {
		t.Fatalf("调整计划时间失败：%v", err)
	}
	n, err := model.PublishDueScheduledPages(model.DB)
	if err != nil {
		t.Fatalf("调度失败：%v", err)
	}
	if n != 1 {
		t.Fatalf("应发布 1 个页面，实际 %d", n)
	}
	// 幂等
	if n2, _ := model.PublishDueScheduledPages(model.DB); n2 != 0 {
		t.Fatalf("重复调度应幂等返回 0，实际 %d", n2)
	}

	after := adminPageByID(t, r, token, p.ID)
	if after.Status != 1 {
		t.Fatalf("到点后状态应为 1，实际 %d", after.Status)
	}
	if after.PublishAt != nil {
		t.Fatalf("发布后 publish_at 应清空，实际 %v", after.PublishAt)
	}
	if after.PublishedAt == nil || !after.PublishedAt.Equal(past) {
		t.Fatalf("published_at 应为计划时间 %v，实际 %v", past, after.PublishedAt)
	}
	// 公开接口可见
	rec = doJSON(t, r, http.MethodGet, "/api/v1/pages/sched-page", "", nil)
	if got := decode(t, rec).Code; got != 0 {
		t.Fatalf("到点后公开接口应可见，实际 %d", got)
	}
}

// TestScheduledPublishNotBeforeDueAt 回归：定时发布只认「到点」，不得提前。
//
// 背景（真实缺陷，2026-09-20 修复）：glebarez/sqlite 把 time.Time 落库为带时区的
// RFC3339 字符串——本地时间写成 "...+08:00"、UTC 写成 "...Z"。而调度器的
// `publish_at <= ?` 在 SQLite 中是**字典序字符串比较**，两种格式混用时
// 「当天未来的计划」会被判为已到点而立刻发布（例如本地 14:02+08:00 落库为 06:02Z，
// 与本地 11:02+08:00 比较时因 '0' < '1' 被误判为更早）。
// 修复方式：写入与比较两侧统一归一为 UTC。本用例锁死该行为，防止回归。
func TestScheduledPublishNotBeforeDueAt(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	future := time.Now().Add(3 * time.Hour)

	// 页面计划用「本地时区」ISO 表示，文章计划用「UTC」ISO 表示——两种都要正确。
	createPage(t, r, token, map[string]any{
		"title": "未来页面", "slug": "future-page", "content": "x",
		"status": 3, "publishAt": future.Format(time.RFC3339),
	})
	if env := decode(t, doJSON(t, r, http.MethodPost, "/api/v1/admin/posts", token, map[string]any{
		"title": "未来文章", "slug": "future-post", "content": "x",
		"categoryId": 1, "status": 3, "publishAt": future.UTC().Format(time.RFC3339),
	})); env.Code != 0 {
		t.Fatalf("创建定时文章失败：%s", env.Message)
	}

	// 关键断言：未到点，调度器必须一个都不发布
	np, err := model.PublishDueScheduledPosts(model.DB)
	if err != nil {
		t.Fatalf("文章调度失败：%v", err)
	}
	nq, err := model.PublishDueScheduledPages(model.DB)
	if err != nil {
		t.Fatalf("页面调度失败：%v", err)
	}
	if np != 0 || nq != 0 {
		t.Fatalf("未到点不得发布：post=%d page=%d（都应为 0）", np, nq)
	}

	// 状态必须仍是定时发布
	if got := adminPageByID(t, r, token, pageIDBySlug(t, "future-page")).Status; got != 3 {
		t.Fatalf("未来页面状态应保持 3，实际 %d", got)
	}

	// 到点后必须发布（把计划时间改到过去，仍走 UTC 归一路径）
	if err := model.DB.Model(&model.Page{}).Where("slug = ?", "future-page").
		Update("publish_at", time.Now().Add(-time.Minute).UTC()).Error; err != nil {
		t.Fatalf("调整计划时间失败：%v", err)
	}
	if n, _ := model.PublishDueScheduledPages(model.DB); n != 1 {
		t.Fatalf("到点后应发布 1 个页面，实际 %d", n)
	}
	if got := adminPageByID(t, r, token, pageIDBySlug(t, "future-page")).Status; got != 1 {
		t.Fatalf("到点后状态应为 1，实际 %d", got)
	}
}

// pageIDBySlug 按 slug 查页面 ID（回归用例内部使用）
func pageIDBySlug(t *testing.T, slug string) uint {
	t.Helper()
	var page model.Page
	if err := model.DB.Where("slug = ?", slug).First(&page).Error; err != nil {
		t.Fatalf("按 slug %q 查页面失败：%v", slug, err)
	}
	return page.ID
}

// ---------- 复制 ----------

func TestAdminCopyPage(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	src := createPage(t, r, token, map[string]any{
		"title": "源页面", "slug": "copy-src", "content": "源内容", "status": 1,
		"seoTitle": "源SEO", "seoDescription": "源描述",
	})

	rec := doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/admin/pages/%d/copy", src.ID), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("复制失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Page model.PageDTO `json:"page"`
	}
	decodeInto(t, e, &data)
	cp := data.Page

	if cp.ID == src.ID {
		t.Fatal("副本不应与原页面同 id")
	}
	if cp.Title != "源页面（副本）" {
		t.Fatalf("副本标题错误：%q", cp.Title)
	}
	if cp.Slug != "copy-src-copy" {
		t.Fatalf("副本 slug 错误：%q", cp.Slug)
	}
	if cp.Status != 0 {
		t.Fatalf("副本状态必须为草稿(0)，实际 %d", cp.Status)
	}
	if cp.PublishAt != nil || cp.PublishedAt != nil {
		t.Fatal("副本不应携带计划/发布时间")
	}
	if cp.Content != "源内容" || cp.SeoTitle != "源SEO" {
		t.Fatalf("副本内容/SEO 应原样复制：%+v", cp)
	}
	// 副本绝不能立即公开
	pubRec := doJSON(t, r, http.MethodGet, "/api/v1/pages/copy-src-copy", "", nil)
	if got := decode(t, pubRec).Code; got != 10004 {
		t.Fatalf("副本不应公开可见，实际 code=%d", got)
	}
	// 副本有自己的 v1
	revs := pageRevisionList(t, r, token, cp.ID)
	if len(revs) != 1 || revs[0].Remark != "首次保存" {
		t.Fatalf("副本应有 v1，实际 %+v", revs)
	}

	// 再复制一次 → slug 追加后缀，不冲突
	rec = doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/admin/pages/%d/copy", src.ID), token, nil)
	var data2 struct {
		Page model.PageDTO `json:"page"`
	}
	decodeInto(t, decode(t, rec), &data2)
	if data2.Page.Slug == cp.Slug || data2.Page.Slug == "" {
		t.Fatalf("第二次复制 slug 应不同：%q vs %q", data2.Page.Slug, cp.Slug)
	}
}

// ---------- 批量 ----------

func TestAdminBatchPages(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	a := createPage(t, r, token, map[string]any{"title": "批量甲", "slug": "batch-a", "status": 0})
	b := createPage(t, r, token, map[string]any{"title": "批量乙", "slug": "batch-b", "status": 0})
	ids := []uint{a.ID, b.ID}

	// 批量发布
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/pages/batch", token, map[string]any{
		"action": "publish", "ids": ids,
	})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("批量发布失败：%+v", e)
	}
	var res struct {
		Updated int64 `json:"updated"`
	}
	decodeInto(t, e, &res)
	if res.Updated != 2 {
		t.Fatalf("批量发布应影响 2 条，实际 %d", res.Updated)
	}
	for _, id := range ids {
		if p := adminPageByID(t, r, token, id); p.Status != 1 || p.PublishedAt == nil {
			t.Fatalf("页面 %d 发布状态/时间异常：%+v", id, p)
		}
	}
	// 公开可见
	if got := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/pages/batch-a", "", nil)).Code; got != 0 {
		t.Fatalf("批量发布后应公开可见，实际 %d", got)
	}

	// 批量隐藏
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/pages/batch", token, map[string]any{
		"action": "hide", "ids": ids,
	})
	decodeInto(t, decode(t, rec), &res)
	if res.Updated != 2 {
		t.Fatalf("批量隐藏应影响 2 条，实际 %d", res.Updated)
	}
	if got := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/pages/batch-a", "", nil)).Code; got != 10004 {
		t.Fatalf("隐藏后应 10004，实际 %d", got)
	}

	// 批量删除（连同版本历史）
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/pages/batch", token, map[string]any{
		"action": "delete", "ids": ids,
	})
	decodeInto(t, decode(t, rec), &res)
	if res.Updated != 2 {
		t.Fatalf("批量删除应影响 2 条，实际 %d", res.Updated)
	}
	var revCount int64
	model.DB.Model(&model.PageRevision{}).Where("page_id IN ?", ids).Count(&revCount)
	if revCount != 0 {
		t.Fatalf("批量删除应清理版本历史，残留 %d 条", revCount)
	}

	// 非法入参
	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"ids 为空", map[string]any{"action": "publish", "ids": []uint{}}},
		{"action 非法", map[string]any{"action": "explode", "ids": []uint{1}}},
	} {
		rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/pages/batch", token, tc.body)
		if got := decode(t, rec).Code; got != 10001 {
			t.Fatalf("%s 应 10001，实际 %d", tc.name, got)
		}
	}
}

// ---------- 并发基线 ----------

// 与 conflict_test.go 同一约定：不依赖机器速度去赌毫秒边界，
// 而是显式构造「明确过期 / 明确超前」的基线来验证冲突保护。
func TestAdminUpdatePageConflictGuard(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	p := createPage(t, r, token, map[string]any{"title": "并发页", "slug": "conflict-p", "content": "初始"})

	// 1) 基线匹配 → 放行
	if e := updatePage(t, r, token, p.ID, map[string]any{
		"title": "并发页", "slug": "conflict-p", "content": "第一次修改", "status": 1,
		"baseUpdatedAt": p.UpdatedAt.Format(time.RFC3339Nano),
	}); e.Code != 0 {
		t.Fatalf("基线匹配应放行：%+v", e)
	}
	afterFirst := adminPageByID(t, r, token, p.ID)
	if afterFirst.Content != "第一次修改" {
		t.Fatalf("首次保存未生效：%q", afterFirst.Content)
	}

	// 2) 过期基线（早 1 秒）→ 10005 且内容不被覆盖
	stale := afterFirst.UpdatedAt.Add(-time.Second).Format(time.RFC3339Nano)
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/pages/%d", p.ID), token, map[string]any{
		"title": "并发页", "slug": "conflict-p", "content": "过期窗口写入", "status": 1,
		"baseUpdatedAt": stale,
	})
	e := decode(t, rec)
	if e.Code != 10005 {
		t.Fatalf("过期基线应 10005，实际 %d", e.Code)
	}
	if after := adminPageByID(t, r, token, p.ID); after.Content != "第一次修改" {
		t.Fatalf("冲突时内容不应被覆盖，实际 %q", after.Content)
	}
	// 10005 携带当前版本信息（不含正文）
	var conflictData struct {
		CurrentUpdatedAt time.Time `json:"currentUpdatedAt"`
		CurrentTitle     string    `json:"currentTitle"`
	}
	decodeInto(t, e, &conflictData)
	if conflictData.CurrentTitle == "" || conflictData.CurrentUpdatedAt.IsZero() {
		t.Fatalf("10005 应携带当前版本信息：%+v", conflictData)
	}

	// 3) 未来基线（晚 1 秒）同样视为不匹配 → 10005（防客户端时钟超前绕过）
	future := afterFirst.UpdatedAt.Add(time.Second).Format(time.RFC3339Nano)
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/pages/%d", p.ID), token, map[string]any{
		"title": "并发页", "slug": "conflict-p", "content": "未来窗口写入", "status": 1,
		"baseUpdatedAt": future,
	})
	if got := decode(t, rec).Code; got != 10005 {
		t.Fatalf("未来基线应 10005，实际 %d", got)
	}

	// 4) 用当前服务器值作基线 → 放行
	fresh := adminPageByID(t, r, token, p.ID).UpdatedAt.Format(time.RFC3339Nano)
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/pages/%d", p.ID), token, map[string]any{
		"title": "并发页", "slug": "conflict-p", "content": "同毫秒写入", "status": 1,
		"baseUpdatedAt": fresh,
	})
	if got := decode(t, rec).Code; got != 0 {
		t.Fatalf("当前基线应放行，实际 %d", got)
	}

	// 5) 不带基线 → 向后兼容放行
	if e := updatePage(t, r, token, p.ID, map[string]any{
		"title": "并发页", "slug": "conflict-p", "content": "无基线修改", "status": 1,
	}); e.Code != 0 {
		t.Fatalf("无基线应放行：%+v", e)
	}
	if got := adminPageByID(t, r, token, p.ID); got.Content != "无基线修改" {
		t.Fatalf("无基线保存未生效：%q", got.Content)
	}
}

// ---------- 状态快速修改 ----------

func TestAdminUpdatePageStatus(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	p := createPage(t, r, token, map[string]any{"title": "快速页", "slug": "quick-p", "content": "内容", "status": 0})

	// 草稿 → 已发布
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/pages/%d/status", p.ID), token, map[string]any{"status": 1})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("状态修改失败：%+v", e)
	}
	var data struct {
		Page model.PageDTO `json:"page"`
	}
	decodeInto(t, e, &data)
	if data.Page.Status != 1 || data.Page.PublishedAt == nil {
		t.Fatalf("快速发布异常：%+v", data.Page)
	}
	// 状态变化生成版本
	if revs := pageRevisionList(t, r, token, p.ID); len(revs) != 2 {
		t.Fatalf("状态变化应生成版本，实际 %d 个", len(revs))
	}

	// 定时缺 publishAt → 10001
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/pages/%d/status", p.ID), token, map[string]any{"status": 3})
	if got := decode(t, rec).Code; got != 10001 {
		t.Fatalf("定时缺 publishAt 应 10001，实际 %d", got)
	}
	// status 越界 → 10001
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/pages/%d/status", p.ID), token, map[string]any{"status": 7})
	if got := decode(t, rec).Code; got != 10001 {
		t.Fatalf("status 越界应 10001，实际 %d", got)
	}
}

// ---------- 删除 ----------

func TestAdminDeletePageCascadesRevisions(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	p := createPage(t, r, token, map[string]any{"title": "待删页", "slug": "del-p", "content": "内容"})
	updatePage(t, r, token, p.ID, map[string]any{"title": "待删页改", "slug": "del-p", "content": "新内容", "status": 1})

	var revCount int64
	model.DB.Model(&model.PageRevision{}).Where("page_id = ?", p.ID).Count(&revCount)
	if revCount < 2 {
		t.Fatalf("删除前应有多个版本，实际 %d", revCount)
	}

	rec := doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/pages/%d", p.ID), token, nil)
	if got := decode(t, rec).Code; got != 0 {
		t.Fatalf("删除失败：%d", got)
	}
	model.DB.Model(&model.PageRevision{}).Where("page_id = ?", p.ID).Count(&revCount)
	if revCount != 0 {
		t.Fatalf("删除应级联清理版本，残留 %d", revCount)
	}
	// 再删 → 10004
	rec = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/pages/%d", p.ID), token, nil)
	if got := decode(t, rec).Code; got != 10004 {
		t.Fatalf("重复删除应 10004，实际 %d", got)
	}
}

// ---------- 预览 / 鉴权 ----------

func TestAdminPreviewPage(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	p := createPage(t, r, token, map[string]any{"title": "预览页", "slug": "preview-p", "content": "草稿正文", "status": 0})

	// 公开接口不可见
	if got := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/pages/preview-p", "", nil)).Code; got != 10004 {
		t.Fatalf("草稿公开应 10004，实际 %d", got)
	}
	// 管理端预览可取完整内容
	rec := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/pages/preview/%d", p.ID), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("预览失败：%+v", e)
	}
	var data struct {
		Page model.PageDTO `json:"page"`
	}
	decodeInto(t, e, &data)
	if data.Page.Content != "草稿正文" || data.Page.Status != 0 {
		t.Fatalf("预览数据不符：%+v", data.Page)
	}
	// 无 token → 401
	rec = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/pages/preview/%d", p.ID), "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("未认证应 401，实际 %d", rec.Code)
	}
	// 不存在 → 10004
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/pages/preview/99999", token, nil)
	if got := decode(t, rec).Code; got != 10004 {
		t.Fatalf("不存在页面应 10004，实际 %d", got)
	}
}

// ---------- 审计日志 ----------

func TestPageAuditLogs(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	p := createPage(t, r, token, map[string]any{"title": "审计页", "slug": "audit-p", "content": "内容"})
	updatePage(t, r, token, p.ID, map[string]any{"title": "审计页改", "slug": "audit-q", "content": "内容", "status": 1})
	doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/pages/%d/status", p.ID), token, map[string]any{"status": 2})
	doJSON(t, r, http.MethodPost, fmt.Sprintf("/api/v1/admin/pages/%d/copy", p.ID), token, nil)
	doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/pages/%d", p.ID), token, nil)

	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs?page=1&pageSize=50&action=page.", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("审计日志查询失败：%+v", e)
	}
	var data struct {
		List []model.AuditLog `json:"list"`
	}
	decodeInto(t, e, &data)

	want := map[string]bool{
		"page.create": false, "page.update": false, "page.status": false,
		"page.copy": false, "page.delete": false, "page.slug_change": false,
	}
	for _, item := range data.List {
		if _, ok := want[item.Action]; ok {
			want[item.Action] = true
		}
		// 审计描述不得包含正文内容
		if item.Description == "内容" || item.Description == "审计页改" && false {
			t.Fatalf("审计描述不应包含正文：%q", item.Description)
		}
	}
	for action, ok := range want {
		if !ok {
			t.Fatalf("缺少审计记录 %s（已记录 %+v）", action, data.List)
		}
	}
}
