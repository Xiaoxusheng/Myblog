package main

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"myblog/server/model"
)

// 公开列表：分页结构、置顶优先、草稿不可见
func TestPublicPostListPaginationAndTop(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)
	base := baseTotal(t, app) // 种子已发布 3 篇

	a := createPost(t, app, token, map[string]any{"title": "置顶文章A", "isTop": true, "status": 1})
	createPost(t, app, token, map[string]any{"title": "普通文章B", "status": 1})
	createPost(t, app, token, map[string]any{"title": "普通文章C", "status": 1})
	createPost(t, app, token, map[string]any{"title": "普通文章D", "status": 1})
	createPost(t, app, token, map[string]any{"title": "草稿E", "status": 0})

	// 第一页：置顶优先，其后按发布时间倒序（D 最新）
	page1 := publicPostList(t, app, "?page=1&pageSize=2")
	if page1.Total != base+4 {
		t.Fatalf("总数错误：期望 %d，实际 %d（草稿不应计入）", base+4, page1.Total)
	}
	if page1.Page != 1 || page1.PageSize != 2 {
		t.Fatalf("分页回显错误：page=%d pageSize=%d", page1.Page, page1.PageSize)
	}
	if len(page1.List) != 2 || page1.List[0].ID != a.ID || !page1.List[0].IsTop {
		t.Fatalf("置顶文章应排第一，实际：%+v", page1.List)
	}
	// 第二页
	page2 := publicPostList(t, app, "?page=2&pageSize=2")
	if len(page2.List) != 2 {
		t.Fatalf("第二页应 2 条，实际 %d", len(page2.List))
	}
	// 列表项不含 content
	for _, p := range append(page1.List, page2.List...) {
		if p.Category != nil && p.Category.Name == "" {
			t.Fatal("category.name 不应为空")
		}
	}
}

// keyword 过滤（含特殊字符 % _ \ 不报错）
func TestPublicPostKeywordFilter(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	createPost(t, app, token, map[string]any{
		"title": "独特关键词干草堆", "content": "正文里出现神秘词针needle在这里", "status": 1,
	})

	// 命中标题
	data := publicPostList(t, app, "?keyword=干草堆")
	if data.Total != 1 {
		t.Fatalf("关键词应命中 1 篇，实际 %d", data.Total)
	}
	// 命中内容
	data = publicPostList(t, app, "?keyword=needle")
	if data.Total != 1 {
		t.Fatalf("内容关键词应命中 1 篇，实际 %d", data.Total)
	}
	// 特殊字符：只作字面量处理，不报错
	for _, kw := range []string{"%", "_", `a\b`, "100%", "__"} {
		rec := doJSON(t, app, http.MethodGet, "/api/v1/posts?keyword="+url.QueryEscape(kw), "", nil)
		if e := decode(t, rec); e.Code != 0 {
			t.Fatalf("特殊字符 keyword=%q 应正常返回 code 0，实际 %d（%s）", kw, e.Code, e.Message)
		}
	}
	// % 单独作为关键词不应把所有文章都查出来（已转义为字面量）
	data = publicPostList(t, app, "?keyword=%25")
	if data.Total >= 1 {
		t.Fatalf("转义后的 %% 不应命中任何文章，实际 %d", data.Total)
	}
}

// 创建文章（带标签）→ 详情可见 → 浏览量 +1 → 点赞递增
func TestPostCreateDetailViewChildLike(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	post := createPost(t, app, token, map[string]any{
		"title":   "接口流程验证文章",
		"summary": "用于验证创建→详情→浏览量→点赞",
		"content": "# 标题\n\n正文内容，用于详情断言。",
		"tags":    []string{"测试标签", "Go"},
		"status":  1,
	})
	if post.Slug != fmt.Sprintf("post-%d", post.ID) {
		t.Fatalf("空 slug 应自动生成为 post-{id}，实际 %q", post.Slug)
	}
	if len(post.Tags) != 2 {
		t.Fatalf("标签数错误：%+v", post.TagNames)
	}
	tagSet := map[string]bool{}
	for _, n := range post.TagNames {
		tagSet[n] = true
	}
	if !tagSet["测试标签"] || !tagSet["Go"] {
		t.Fatalf("标签未正确保存：%+v", post.TagNames)
	}
	if post.PublishedAt == nil {
		t.Fatal("首次发布应写入 publishedAt")
	}

	// 详情（slug 访问）
	rec := doJSON(t, app, http.MethodGet, "/api/v1/posts/"+post.Slug, "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("详情失败：%s", e.Message)
	}
	var detail struct {
		Post struct {
			ID        uint   `json:"id"`
			Content   string `json:"content"`
			ViewCount int    `json:"viewCount"`
			LikeCount int    `json:"likeCount"`
			Tags      []struct {
				Name string `json:"name"`
			} `json:"tags"`
		} `json:"post"`
		Prev    map[string]any `json:"prev"`
		Next    map[string]any `json:"next"`
		Related []struct {
			ID uint `json:"id"`
		} `json:"related"`
	}
	decodeInto(t, e, &detail)
	if detail.Post.ID != post.ID || detail.Post.Content != "# 标题\n\n正文内容，用于详情断言。" {
		t.Fatalf("详情内容不符：%+v", detail.Post)
	}
	if detail.Post.ViewCount != 1 {
		t.Fatalf("首次访问后浏览量应为 1，实际 %d", detail.Post.ViewCount)
	}
	if len(detail.Post.Tags) != 2 {
		t.Fatalf("详情标签数错误：%d", len(detail.Post.Tags))
	}

	// 再访问一次 → 浏览量 2
	rec = doJSON(t, app, http.MethodGet, "/api/v1/posts/"+post.Slug, "", nil)
	e = decode(t, rec)
	decodeInto(t, e, &detail)
	if detail.Post.ViewCount != 2 {
		t.Fatalf("二次访问浏览量应为 2，实际 %d", detail.Post.ViewCount)
	}

	// 点赞两次
	for want := 1; want <= 2; want++ {
		rec := doJSON(t, app, http.MethodPost, "/api/v1/posts/"+post.Slug+"/like", "", nil)
		e := decode(t, rec)
		if e.Code != 0 {
			t.Fatalf("点赞失败：%s", e.Message)
		}
		var like struct {
			LikeCount int `json:"likeCount"`
		}
		decodeInto(t, e, &like)
		if like.LikeCount != want {
			t.Fatalf("第 %d 次点赞后 likeCount 应为 %d，实际 %d", want, want, like.LikeCount)
		}
	}

	// 详情可见点赞数
	rec = doJSON(t, app, http.MethodGet, "/api/v1/posts/"+post.Slug, "", nil)
	e = decode(t, rec)
	decodeInto(t, e, &detail)
	if detail.Post.LikeCount != 2 {
		t.Fatalf("详情 likeCount 应为 2，实际 %d", detail.Post.LikeCount)
	}
}

// 详情：数字 id 访问 + prev/next 顺序（基于种子文章的发布时间）
func TestPostDetailByIdAndPrevNext(t *testing.T) {
	app := newTestApp(t)

	// 通过公开列表找种子文章 vue3(-48h)，其前是 gin(-72h)，其后是 go(-24h)
	list := publicPostList(t, app, "?sort=newest&pageSize=50")
	var target model.PostSummaryDTO
	for _, p := range list.List {
		if p.Slug == "vue3-composition-api" {
			target = p
		}
	}
	if target.ID == 0 {
		t.Fatal("种子文章 vue3-composition-api 未找到")
	}

	// 数字 id 访问详情
	rec := doJSON(t, app, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", target.ID), "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("按 id 访问详情失败：%s", e.Message)
	}
	var data struct {
		Post struct {
			ID uint `json:"id"`
		} `json:"post"`
		Prev *struct {
			Slug string `json:"slug"`
		} `json:"prev"`
		Next *struct {
			Slug string `json:"slug"`
		} `json:"next"`
	}
	decodeInto(t, e, &data)
	if data.Post.ID != target.ID {
		t.Fatalf("按 id 取到的文章不符：%d", data.Post.ID)
	}
	if data.Prev == nil || data.Prev.Slug != "gin-blog-backend" {
		t.Fatalf("prev 应为较早发布的 gin-blog-backend：%+v", data.Prev)
	}
	if data.Next == nil || data.Next.Slug != "go-concurrency" {
		t.Fatalf("next 应为较晚发布的 go-concurrency：%+v", data.Next)
	}
}

// slug 自动生成：空 → post-{id}；重复 → 追加 -{id}
func TestSlugAutoGeneration(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	first := createPost(t, app, token, map[string]any{"title": "_slug 文章一", "slug": "my-custom-slug"})
	if first.Slug != "my-custom-slug" {
		t.Fatalf("自定义 slug 应保留，实际 %q", first.Slug)
	}
	second := createPost(t, app, token, map[string]any{"title": "slug 文章二", "slug": "my-custom-slug"})
	if second.Slug == "my-custom-slug" || second.Slug == "" {
		t.Fatalf("重复 slug 应自动改写，实际 %q", second.Slug)
	}
	want := fmt.Sprintf("my-custom-slug-%d", second.ID)
	if second.Slug != want {
		t.Fatalf("重复 slug 应追加 -id（期望 %s），实际 %s", want, second.Slug)
	}
	empty := createPost(t, app, token, map[string]any{"title": "slug 文章三"})
	if empty.Slug != fmt.Sprintf("post-%d", empty.ID) {
		t.Fatalf("空 slug 应为 post-{id}，实际 %q", empty.Slug)
	}
}

// 管理端列表：commentCount / status 过滤 / 管理详情
func TestAdminPostListAndDetail(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	draft := createPost(t, app, token, map[string]any{"title": "管理端草稿", "status": 0})

	// 状态过滤
	rec := doJSON(t, app, http.MethodGet, "/api/v1/admin/posts?status=0", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("管理列表失败：%s", e.Message)
	}
	var data struct {
		List []model.AdminPostItemDTO `json:"list"`
	}
	decodeInto(t, e, &data)
	found := false
	for _, p := range data.List {
		if p.ID == draft.ID {
			found = true
			if p.Content == "" || p.CategoryID != 0 || p.TagNames == nil {
				t.Fatalf("AdminPostItem 字段缺失：%+v", p)
			}
		}
	}
	if !found {
		t.Fatal("status=0 过滤应包含草稿")
	}

	// 管理详情
	rec = doJSON(t, app, http.MethodGet, fmt.Sprintf("/api/v1/admin/posts/%d", draft.ID), token, nil)
	e = decode(t, rec)
	var got struct {
		Post model.AdminPostItemDTO `json:"post"`
	}
	decodeInto(t, e, &got)
	if got.Post.ID != draft.ID || got.Post.Status != 0 {
		t.Fatalf("管理详情错误：%+v", got.Post)
	}
}

// 更新文章 + 状态切换（首次发布写 published_at）
func TestAdminPostUpdateAndStatus(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	post := createPost(t, app, token, map[string]any{"title": "更新前", "status": 0})
	if post.PublishedAt != nil {
		t.Fatal("草稿不应有 publishedAt")
	}

	// 全量更新为已发布
	rec := doJSON(t, app, http.MethodPut, fmt.Sprintf("/api/v1/admin/posts/%d", post.ID), token,
		map[string]any{
			"title": "更新后", "slug": "", "summary": "新摘要", "content": "新内容",
			"cover": "", "categoryId": 0, "tags": []string{"Gin"}, "status": 1, "isTop": false,
		})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("更新失败：%s", e.Message)
	}
	var updated struct {
		Post model.AdminPostItemDTO `json:"post"`
	}
	decodeInto(t, e, &updated)
	if updated.Post.Title != "更新后" || updated.Post.Status != 1 || updated.Post.PublishedAt == nil {
		t.Fatalf("更新结果错误：title=%s status=%d publishedAt=%v",
			updated.Post.Title, updated.Post.Status, updated.Post.PublishedAt)
	}

	// 隐藏：公开详情 10004
	rec = doJSON(t, app, http.MethodPut, fmt.Sprintf("/api/v1/admin/posts/%d/status", post.ID), token,
		map[string]any{"status": 2})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("切换状态失败：%s", e.Message)
	}
	rec = doJSON(t, app, http.MethodGet, "/api/v1/posts/"+updated.Post.Slug, "", nil)
	if e := decode(t, rec); e.Code != 10004 {
		t.Fatalf("隐藏文章公开详情应 10004，实际 %d", e.Code)
	}
}
