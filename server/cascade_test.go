package main

import (
	"fmt"
	"net/http"
	"testing"

	"myblog/server/model"
)

// 删文章级联：post_tags 与评论一并删除
func TestDeletePostCascades(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)
	post := createPost(t, app, token, map[string]any{
		"title": "级联删除文章", "status": 1, "tags": []string{"级联标签A", "级联标签B"},
	})

	// 游客评论 + 通过
	rec := doJSON(t, app, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "",
		map[string]any{"nickname": "游客", "email": "g@e.com", "content": "评论内容"})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("评论失败：%s", e.Message)
	}
	var created struct {
		Comment struct {
			ID uint `json:"id"`
		} `json:"comment"`
	}
	decodeInto(t, e, &created)
	rec = doJSON(t, app, http.MethodPut, fmt.Sprintf("/api/v1/admin/comments/%d/status", created.Comment.ID), token,
		map[string]any{"status": 1})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("审核失败：%s", e.Message)
	}

	// 前置确认
	var tagN, cmN int64
	model.DB.Model(&model.PostTag{}).Where("post_id = ?", post.ID).Count(&tagN)
	model.DB.Model(&model.Comment{}).Where("post_id = ?", post.ID).Count(&cmN)
	if tagN != 2 || cmN != 1 {
		t.Fatalf("前置数据错误：post_tags=%d comments=%d", tagN, cmN)
	}

	// 删除文章
	rec = doJSON(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/posts/%d", post.ID), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除文章失败：%s", e.Message)
	}

	model.DB.Model(&model.PostTag{}).Where("post_id = ?", post.ID).Count(&tagN)
	model.DB.Model(&model.Comment{}).Where("post_id = ?", post.ID).Count(&cmN)
	if tagN != 0 || cmN != 0 {
		t.Fatalf("级联删除失败：post_tags=%d comments=%d", tagN, cmN)
	}

	// 公开详情 10004
	rec = doJSON(t, app, http.MethodGet, "/api/v1/posts/"+post.Slug, "", nil)
	if e := decode(t, rec); e.Code != 10004 {
		t.Fatalf("删除后详情应 10004，实际 %d", e.Code)
	}
}

// 删分类：其下文章 categoryId 置 0
func TestDeleteCategoryResetsPosts(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	rec := doJSON(t, app, http.MethodPost, "/api/v1/admin/categories", token,
		map[string]any{"name": "临时分类", "description": "将被删除"})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建分类失败：%s", e.Message)
	}
	var category struct {
		ID uint `json:"id"`
	}
	decodeInto(t, e, &category)

	post := createPost(t, app, token, map[string]any{"title": "分类归属文章", "status": 1, "categoryId": category.ID})
	if post.CategoryID != category.ID {
		t.Fatalf("文章分类未生效：%d", post.CategoryID)
	}

	rec = doJSON(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/categories/%d", category.ID), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除分类失败：%s", e.Message)
	}

	var p model.Post
	if err := model.DB.First(&p, post.ID).Error; err != nil {
		t.Fatalf("文章应仍存在：%v", err)
	}
	if p.CategoryID != 0 {
		t.Fatalf("删除分类后 categoryId 应置 0，实际 %d", p.CategoryID)
	}
}

// 删标签：post_tags 同步清理
func TestDeleteTagClearsPostTags(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	rec := doJSON(t, app, http.MethodPost, "/api/v1/admin/tags", token, map[string]any{"name": "临时标签"})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建标签失败：%s", e.Message)
	}
	var tag struct {
		ID uint `json:"id"`
	}
	decodeInto(t, e, &tag)

	post := createPost(t, app, token, map[string]any{"title": "标签归属文章", "status": 1, "tags": []string{"临时标签"}})
	var n int64
	model.DB.Model(&model.PostTag{}).Where("tag_id = ?", tag.ID).Count(&n)
	if n != 1 {
		t.Fatalf("post_tags 应有 1 行，实际 %d", n)
	}

	rec = doJSON(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/tags/%d", tag.ID), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除标签失败：%s", e.Message)
	}
	model.DB.Model(&model.PostTag{}).Where("tag_id = ?", tag.ID).Count(&n)
	if n != 0 {
		t.Fatalf("post_tags 未清理，剩 %d 行", n)
	}
	_ = post
}

// 分类/标签名称唯一
func TestTaxonomyUniqueName(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	rec := doJSON(t, app, http.MethodPost, "/api/v1/admin/categories", token, map[string]any{"name": "重复分类"})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("创建分类失败：%s", e.Message)
	}
	rec = doJSON(t, app, http.MethodPost, "/api/v1/admin/categories", token, map[string]any{"name": "重复分类"})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("重复分类名应 10001，实际 %d", e.Code)
	}

	rec = doJSON(t, app, http.MethodPost, "/api/v1/admin/tags", token, map[string]any{"name": "重复标签"})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("创建标签失败：%s", e.Message)
	}
	rec = doJSON(t, app, http.MethodPost, "/api/v1/admin/tags", token, map[string]any{"name": "重复标签"})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("重复标签名应 10001，实际 %d", e.Code)
	}
}
