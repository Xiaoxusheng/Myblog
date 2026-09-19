package main

// 文章版本历史（契约 #50-52）与定时发布（status=3 + 调度器）测试

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"myblog/server/model"
)

// ---------- 测试助手 ----------

// updatePostHelper 全量更新文章（契约 #19 为全量语义，默认值与 createPost 对齐），返回更新后文章
func updatePostHelper(t *testing.T, r http.Handler, token string, id uint, fields map[string]any) model.AdminPostItemDTO {
	t.Helper()
	payload := map[string]any{
		"title":   "默认标题",
		"slug":    "",
		"summary": "默认摘要",
		"content": "默认正文内容",
		"cover":   "",
		"status":  1,
		"isTop":   false,
		"tags":    []string{},
	}
	for k, v := range fields {
		payload[k] = v
	}
	rec := doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/posts/%d", id), token, payload)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("更新文章失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Post model.AdminPostItemDTO `json:"post"`
	}
	decodeInto(t, e, &data)
	return data.Post
}

type revisionListData struct {
	List []struct {
		ID      uint   `json:"id"`
		PostID  uint   `json:"postId"`
		Version int    `json:"version"`
		Remark  string `json:"remark"`
	} `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

func adminRevisionList(t *testing.T, r http.Handler, token string, postID uint) revisionListData {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/posts/%d/revisions", postID), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("版本列表失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data revisionListData
	decodeInto(t, e, &data)
	return data
}

func adminRevisionDetail(t *testing.T, r http.Handler, token string, postID uint, version int) model.PostRevision {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/posts/%d/revisions/%d", postID, version), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("版本详情失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Revision model.PostRevision `json:"revision"`
	}
	decodeInto(t, e, &data)
	return data.Revision
}

func adminRestoreRevision(t *testing.T, r http.Handler, token string, postID uint, version int) model.AdminPostItemDTO {
	t.Helper()
	rec := doJSON(t, r, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/posts/%d/revisions/%d/restore", postID, version), token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("恢复版本失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Post model.AdminPostItemDTO `json:"post"`
	}
	decodeInto(t, e, &data)
	return data.Post
}

func adminStatsData(t *testing.T, r http.Handler, token string) struct {
	ScheduledCount int64 `json:"scheduledCount"`
	ScheduledPosts []struct {
		ID        uint       `json:"id"`
		Title     string     `json:"title"`
		PublishAt *time.Time `json:"publishAt"`
	} `json:"scheduledPosts"`
} {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/stats", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("stats 失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		ScheduledCount int64 `json:"scheduledCount"`
		ScheduledPosts []struct {
			ID        uint       `json:"id"`
			Title     string     `json:"title"`
			PublishAt *time.Time `json:"publishAt"`
		} `json:"scheduledPosts"`
	}
	decodeInto(t, e, &data)
	return data
}

// ---------- 版本历史 ----------

func TestRevisionLifecycle(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	// 创建 → v1 首次保存
	post := createPost(t, r, token, map[string]any{"title": "标题一", "content": "内容一"})
	list := adminRevisionList(t, r, token, post.ID)
	if list.Total != 1 || len(list.List) != 1 {
		t.Fatalf("创建后应有 1 个版本，got total=%d", list.Total)
	}
	if list.List[0].Version != 1 || list.List[0].Remark != "首次保存" {
		t.Fatalf("v1 应为 首次保存，got v%d %q", list.List[0].Version, list.List[0].Remark)
	}

	// 改标题（内容保持不变）→ v2 修改标题
	updatePostHelper(t, r, token, post.ID, map[string]any{"title": "标题二", "content": "内容一"})
	list = adminRevisionList(t, r, token, post.ID)
	if list.Total != 2 || list.List[0].Remark != "修改标题" {
		t.Fatalf("改标题后 total=2 且最新 remark=修改标题，got total=%d remark=%q",
			list.Total, list.List[0].Remark)
	}

	// 原样再存（无变化）→ 不生成版本
	updatePostHelper(t, r, token, post.ID, map[string]any{"title": "标题二", "content": "内容一"})
	if list := adminRevisionList(t, r, token, post.ID); list.Total != 2 {
		t.Fatalf("无变化保存不应生成版本，got total=%d", list.Total)
	}

	// auto 保存（内容变化，距最新版本 < 120s）→ 防抖不生成版本
	updatePostHelper(t, r, token, post.ID, map[string]any{"title": "标题二", "content": "内容二", "auto": true})
	if list := adminRevisionList(t, r, token, post.ID); list.Total != 2 {
		t.Fatalf("auto 防抖期内保存不应生成版本，got total=%d", list.Total)
	}

	// 手动保存（内容变化）→ v3 修改正文
	updatePostHelper(t, r, token, post.ID, map[string]any{"title": "标题二", "content": "内容三"})
	list = adminRevisionList(t, r, token, post.ID)
	if list.Total != 3 || list.List[0].Remark != "修改正文" {
		t.Fatalf("改正文后 total=3 且最新 remark=修改正文，got total=%d remark=%q",
			list.Total, list.List[0].Remark)
	}

	// 版本详情：v1 快照为创建时内容
	v1 := adminRevisionDetail(t, r, token, post.ID, 1)
	if v1.Content != "内容一" || v1.Title != "标题一" || v1.Version != 1 {
		t.Fatalf("v1 快照不符：title=%q content=%q", v1.Title, v1.Content)
	}

	// 不存在版本 → 10004；不存在文章的版本列表 → 10004
	if e := decode(t, doJSON(t, r, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/posts/%d/revisions/99", post.ID), token, nil)); e.Code != 10004 {
		t.Fatalf("不存在版本应 10004，got code=%d", e.Code)
	}
	if e := decode(t, doJSON(t, r, http.MethodGet,
		"/api/v1/admin/posts/999999/revisions", token, nil)); e.Code != 10004 {
		t.Fatalf("不存在文章应 10004，got code=%d", e.Code)
	}
}

func TestRevisionRestore(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	// 已发布文章：创建（标题一）→ 改标题 → 恢复 v1
	post := createPost(t, r, token, map[string]any{"title": "标题一", "content": "内容一", "status": 1})
	updatePostHelper(t, r, token, post.ID, map[string]any{"title": "标题二", "content": "内容一"})

	restored := adminRestoreRevision(t, r, token, post.ID, 1)
	if restored.Title != "标题一" {
		t.Fatalf("恢复后标题应为 标题一，got %q", restored.Title)
	}
	// 恢复不改发布状态（契约 #52）
	if restored.Status != model.PostPublished {
		t.Fatalf("恢复不应改变发布状态，got status=%d", restored.Status)
	}

	// 当前内容与最新版本（v2）一致 → 跳过「恢复前快照」，仅生成「恢复自 v1」
	// 版本序列：v1 首次保存 / v2 修改标题 / v3 恢复自 v1
	list := adminRevisionList(t, r, token, post.ID)
	if list.Total != 3 {
		t.Fatalf("恢复后应有 3 个版本，got %d", list.Total)
	}
	if list.List[0].Remark != "恢复自 v1" {
		t.Fatalf("最新版本 remark 应为 恢复自 v1，got %q", list.List[0].Remark)
	}

	// 撤销恢复：恢复 v2 → 标题回到 标题二（当前内容与最新版本 v3 一致 → 也跳过恢复前快照）
	undo := adminRestoreRevision(t, r, token, post.ID, 2)
	if undo.Title != "标题二" {
		t.Fatalf("撤销恢复后标题应为 标题二，got %q", undo.Title)
	}
	if list := adminRevisionList(t, r, token, post.ID); list.Total != 4 {
		t.Fatalf("撤销恢复后应有 4 个版本，got %d", list.Total)
	}

	// 恢复前快照分支（白盒）：直接改库模拟「当前内容未落版本」（如存量文章），
	// 此时恢复必须先自动快照当前内容，保证可撤销。
	if err := model.DB.Model(&model.Post{}).Where("id = ?", post.ID).
		Update("title", "直改标题").Error; err != nil {
		t.Fatalf("直改文章失败：%v", err)
	}
	adminRestoreRevision(t, r, token, post.ID, 2)
	list = adminRevisionList(t, r, token, post.ID)
	if list.Total != 6 {
		t.Fatalf("未落版本场景恢复应产生 恢复前快照+恢复自 v2 共 6 个版本，got %d", list.Total)
	}
	if list.List[0].Remark != "恢复自 v2" || list.List[1].Remark != "恢复前快照" {
		t.Fatalf("版本 remark 不符：%q / %q", list.List[0].Remark, list.List[1].Remark)
	}
	// 撤销：恢复「恢复前快照」→ 标题回到 直改标题
	undoAgain := adminRestoreRevision(t, r, token, post.ID, 5)
	if undoAgain.Title != "直改标题" {
		t.Fatalf("撤销后标题应为 直改标题，got %q", undoAgain.Title)
	}

	// 无实际变化（恢复当前内容所在版本）→ 不新增版本
	before := adminRevisionList(t, r, token, post.ID)
	adminRestoreRevision(t, r, token, post.ID, before.List[0].Version)
	if after := adminRevisionList(t, r, token, post.ID); after.Total != before.Total {
		t.Fatalf("无变化恢复不应生成版本，before=%d after=%d", before.Total, after.Total)
	}
}

func TestRestoreSlugConflict(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	// B 的 v1 快照 slug=b-old；随后 B 改 slug 释放 b-old，再让 C 占用 b-old
	postB := createPost(t, r, token, map[string]any{"title": "文章B", "slug": "b-old"})
	updatePostHelper(t, r, token, postB.ID, map[string]any{"title": "文章B", "slug": "b-new"})
	createPost(t, r, token, map[string]any{"title": "文章C", "slug": "b-old"})

	rec := doJSON(t, r, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/posts/%d/revisions/1/restore", postB.ID), token, nil)
	e := decode(t, rec)
	if e.Code != 10001 {
		t.Fatalf("恢复占用 slug 应 10001，got code=%d msg=%s", e.Code, e.Message)
	}
	// 回滚后不应产生版本
	if list := adminRevisionList(t, r, token, postB.ID); list.Total != 2 {
		t.Fatalf("冲突恢复不应生成版本，got total=%d", list.Total)
	}
}

// ---------- 定时发布 ----------

func TestScheduledPublishing(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	baseTotal := baseTotal(t, r)

	// status=3 缺 publishAt → 10001
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/posts", token, map[string]any{
		"title": "缺时间", "content": "内容", "status": 3, "tags": []string{},
	})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("定时发布缺 publishAt 应 10001，got code=%d msg=%s", e.Code, e.Message)
	}

	// 未来计划：不可见、计入统计
	future := createPost(t, r, token, map[string]any{
		"title": "未来发布", "content": "未来内容", "status": 3,
		"publishAt": time.Now().Add(time.Hour),
	})
	if got := publicPostList(t, r, "?page=1&pageSize=50"); got.Total != baseTotal {
		t.Fatalf("定时文章不应出现在公开列表，base=%d got=%d", baseTotal, got.Total)
	}
	if e := decode(t, doJSON(t, r, http.MethodGet,
		"/api/v1/posts/"+future.Slug, "", nil)); e.Code != 10004 {
		t.Fatalf("定时文章详情应 10004，got code=%d", e.Code)
	}
	stats := adminStatsData(t, r, token)
	if stats.ScheduledCount != 1 || len(stats.ScheduledPosts) != 1 || stats.ScheduledPosts[0].ID != future.ID {
		t.Fatalf("stats 计划发布统计不符：%+v", stats.ScheduledPosts)
	}

	// 过期计划：调度器一次扫描后发布，published_at = 计划时间，publish_at 清空
	dueAt := time.Now().Add(-time.Hour)
	due := createPost(t, r, token, map[string]any{
		"title": "到期发布", "content": "到期内容", "status": 3, "publishAt": dueAt,
	})
	n, err := model.PublishDueScheduledPosts(model.DB)
	if err != nil || n != 1 {
		t.Fatalf("调度器应发布 1 篇，got n=%d err=%v", n, err)
	}
	// 幂等：再跑一次为 0
	if n, _ := model.PublishDueScheduledPosts(model.DB); n != 0 {
		t.Fatalf("调度器应幂等，got n=%d", n)
	}
	if got := publicPostList(t, r, "?page=1&pageSize=50"); got.Total != baseTotal+1 {
		t.Fatalf("发布后公开列表应 +1，base=%d got=%d", baseTotal, got.Total)
	}
	// 通过 admin 详情接口验证：已发布、published_at=计划时间、publish_at 已清空
	detailRec := doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/posts/%d", due.ID), token, nil)
	detailEnv := decode(t, detailRec)
	if detailEnv.Code != 0 {
		t.Fatalf("获取文章失败：%s", detailEnv.Message)
	}
	var detail struct {
		Post model.AdminPostItemDTO `json:"post"`
	}
	if err := json.Unmarshal(detailEnv.Data, &detail); err != nil {
		t.Fatalf("解析文章失败：%v", err)
	}
	if detail.Post.Status != model.PostPublished {
		t.Fatalf("到点后应为已发布，got status=%d", detail.Post.Status)
	}
	if detail.Post.PublishedAt == nil || detail.Post.PublishAt != nil {
		t.Fatalf("published_at 应为计划时间且 publish_at 应清空：%+v", detail.Post)
	}
	if diff := detail.Post.PublishedAt.Sub(dueAt); diff < -time.Minute || diff > time.Minute {
		t.Fatalf("published_at 应等于计划时间，diff=%v", diff)
	}

	// #20 状态端点：草稿 → 定时发布
	draft := createPost(t, r, token, map[string]any{"title": "草稿转定时", "status": 0})
	rec = doJSON(t, r, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/posts/%d/status", draft.ID), token,
		map[string]any{"status": 3, "publishAt": time.Now().Add(2 * time.Hour)})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("状态端点转定时失败：code=%d msg=%s", e.Code, e.Message)
	}
	stats = adminStatsData(t, r, token)
	if stats.ScheduledCount != 2 {
		t.Fatalf("转定时后 scheduledCount 应为 2，got %d", stats.ScheduledCount)
	}

	// #20 缺 publishAt → 10001
	rec = doJSON(t, r, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/posts/%d/status", draft.ID), token, map[string]any{"status": 3})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("状态端点缺 publishAt 应 10001，got code=%d", e.Code)
	}
}

// ---------- 级联删除 ----------

func TestDeletePostCascadesRevisions(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	post := createPost(t, r, token, map[string]any{"title": "标题一"})
	updatePostHelper(t, r, token, post.ID, map[string]any{"title": "标题二"})
	if list := adminRevisionList(t, r, token, post.ID); list.Total != 2 {
		t.Fatalf("删除前应有 2 个版本，got %d", list.Total)
	}

	rec := doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/posts/%d", post.ID), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除失败：code=%d", e.Code)
	}

	var count int64
	model.DB.Model(&model.PostRevision{}).Where("post_id = ?", post.ID).Count(&count)
	if count != 0 {
		t.Fatalf("删除文章应级联删除版本，残留 %d 条", count)
	}
}
