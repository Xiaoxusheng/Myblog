package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"myblog/server/model"
)

// 游客评论：创建 → 待审（公开不可见）→ 管理端通过 → 公开树可见
func TestGuestCommentFlow(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)
	post := createPost(t, app, token, map[string]any{"title": "评论流程文章", "status": 1})

	// 游客评论（入参含 parentId 省略、website 选填）
	rec := doJSON(t, app, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "",
		map[string]any{"nickname": "游客甲", "email": "guest@example.com", "content": "写得不错！"})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("游客评论失败：%s", e.Message)
	}
	var created struct {
		Comment struct {
			ID       uint   `json:"id"`
			Nickname string `json:"nickname"`
		} `json:"comment"`
	}
	decodeInto(t, e, &created)
	if created.Comment.ID == 0 || created.Comment.Nickname != "游客甲" {
		t.Fatalf("评论返回体错误：%+v", created.Comment)
	}

	// 待审核：公开列表不可见
	list := publicComments(t, app, post.Slug)
	if len(list) != 0 {
		t.Fatalf("待审评论不应公开可见，实际 %d 条", len(list))
	}

	// 管理端待审列表 1 条，且带 IP
	rec = doJSON(t, app, http.MethodGet, "/api/v1/admin/comments?status=0", token, nil)
	e = decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("管理评论列表失败：%s", e.Message)
	}
	var adminList struct {
		List []struct {
			ID        uint   `json:"id"`
			PostTitle string `json:"postTitle"`
			IP        string `json:"ip"`
			Status    int    `json:"status"`
			Email     string `json:"email"`
		} `json:"list"`
		Total int64 `json:"total"`
	}
	decodeInto(t, e, &adminList)
	if adminList.Total != 1 || len(adminList.List) != 1 {
		t.Fatalf("待审评论应 1 条，实际 %d", adminList.Total)
	}
	if adminList.List[0].PostTitle != "评论流程文章" || adminList.List[0].IP == "" {
		t.Fatalf("CommentAdmin 缺少 postTitle/ip：%+v", adminList.List[0])
	}
	commentID := adminList.List[0].ID

	// 通过
	rec = doJSON(t, app, http.MethodPut, fmt.Sprintf("/api/v1/admin/comments/%d/status", commentID), token,
		map[string]any{"status": 1})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("审核通过失败：%s", e.Message)
	}

	// 公开树可见
	list = publicComments(t, app, post.Slug)
	if len(list) != 1 || list[0].Nickname != "游客甲" {
		t.Fatalf("通过后公开评论错误：%+v", list)
	}
	if list[0].Children == nil || len(list[0].Children) != 0 {
		t.Fatalf("children 应为空数组：%+v", list[0].Children)
	}
}

// 管理员回复：直接通过并出现在公开树（isAdmin=1）；种子评论树结构正确
func TestAdminReplyVisible(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	// 种子文章 gin-blog-backend：张三（顶）+ 管理员回复（子）+ 李四（顶）
	list := publicComments(t, app, "gin-blog-backend")
	if len(list) != 2 {
		t.Fatalf("种子公开评论应为 2 个顶级，实际 %d", len(list))
	}
	// 顶级倒序：李四（后创建）在前
	if list[0].Nickname != "李四" || list[1].Nickname != "张三" {
		t.Fatalf("顶级排序错误：%s, %s", list[0].Nickname, list[1].Nickname)
	}
	if len(list[1].Children) != 1 || !list[1].Children[0].IsAdmin {
		t.Fatalf("张三下应有管理员回复：%+v", list[1].Children)
	}

	// 管理员回复李四的评论
	rec := doJSON(t, app, http.MethodPost, fmt.Sprintf("/api/v1/admin/comments/%d/reply", list[0].ID), token,
		map[string]any{"content": "收到，已安排补充。"})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("管理员回复失败：%s", e.Message)
	}
	var reply struct {
		Comment struct {
			ID        uint   `json:"id"`
			PostID    uint   `json:"postId"`
			PostTitle string `json:"postTitle"`
			ParentID  uint   `json:"parentId"`
			Nickname  string `json:"nickname"`
			Status    int    `json:"status"`
			IsAdmin   bool   `json:"isAdmin"`
		} `json:"comment"`
	}
	decodeInto(t, e, &reply)
	if reply.Comment.ParentID != list[0].ID || reply.Comment.Status != 1 || !reply.Comment.IsAdmin {
		t.Fatalf("回复元数据错误：%+v", reply.Comment)
	}
	if reply.Comment.PostTitle != "使用 Gin 搭建高性能博客后端" {
		t.Fatalf("回复 postTitle 错误：%q", reply.Comment.PostTitle)
	}

	// 公开树：李四下可见回复（无需再审核）
	list = publicComments(t, app, "gin-blog-backend")
	if len(list[0].Children) != 1 || list[0].Children[0].Content != "收到，已安排补充。" {
		t.Fatalf("管理员回复应直接公开可见：%+v", list[0].Children)
	}
}

// 评论关闭 → 20002
func TestCommentDisabled(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)
	post := createPost(t, app, token, map[string]any{"title": "关闭评论的文章", "status": 1})

	// 关闭评论开关
	settings := adminSettings(t, app, token)
	settings["commentEnabled"] = false
	putSettings(t, app, token, settings)

	rec := doJSON(t, app, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "",
		map[string]any{"nickname": "游客", "email": "g@e.com", "content": "试试"})
	if e := decode(t, rec); e.Code != 20002 {
		t.Fatalf("评论关闭应 20002，实际 %d", e.Code)
	}
}

// 评论限流：同 IP 60 秒第 2 次提交 → 20003
func TestCommentRateLimit(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)
	post := createPost(t, app, token, map[string]any{"title": "限流文章", "status": 1})

	body := func(i int) map[string]any {
		return map[string]any{"nickname": fmt.Sprintf("游客%d", i), "email": "g@e.com", "content": "第 n 条"}
	}
	rec := doJSON(t, app, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "", body(1))
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("第一次评论应成功：%s", e.Message)
	}
	rec = doJSON(t, app, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "", body(2))
	if e := decode(t, rec); e.Code != 20003 {
		t.Fatalf("第二次评论应 20003，实际 %d", e.Code)
	}
}

// 评论校验：必填项、内容长度
func TestCommentValidation(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)
	post := createPost(t, app, token, map[string]any{"title": "校验文章", "status": 1})

	// 缺昵称
	rec := doJSON(t, app, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "",
		map[string]any{"nickname": "", "email": "g@e.com", "content": "内容"})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("缺昵称应 10001，实际 %d", e.Code)
	}

	// 内容超 1000 字
	rec = doJSON(t, app, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "",
		map[string]any{"nickname": "游客", "email": "g@e.com", "content": strings.Repeat("长", 1001)})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("超长内容应 10001，实际 %d", e.Code)
	}

	// 回复不存在的评论
	rec = doJSON(t, app, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "",
		map[string]any{"nickname": "游客", "email": "g@e.com", "content": "回复", "parentId": 99999})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("回复不存在评论应 10001，实际 %d", e.Code)
	}
}

// 删除评论连带 children（种子：张三有管理员回复）
func TestDeleteCommentWithChildren(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	list := publicComments(t, app, "gin-blog-backend")
	var zhangsanID uint
	for _, cm := range list {
		if cm.Nickname == "张三" {
			zhangsanID = cm.ID
		}
	}
	if zhangsanID == 0 {
		t.Fatal("种子评论张三未找到")
	}

	rec := doJSON(t, app, http.MethodDelete, fmt.Sprintf("/api/v1/admin/comments/%d", zhangsanID), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除评论失败：%s", e.Message)
	}

	// 张三与其回复都消失
	var count int64
	model.DB.Model(&model.Comment{}).
		Where("post_id = (SELECT id FROM posts WHERE slug = 'gin-blog-backend')").
		Count(&count)
	if count != 1 { // 只剩李四
		t.Fatalf("删除后该文章应剩 1 条评论，实际 %d", count)
	}
}

// ---------- 评论相关助手 ----------

func publicComments(t *testing.T, r http.Handler, slug string) []model.CommentPublicDTO {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/posts/"+slug+"/comments", "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("公开评论列表失败：%s", e.Message)
	}
	var data struct {
		List []model.CommentPublicDTO `json:"list"`
	}
	decodeInto(t, e, &data)
	return data.List
}

func adminSettings(t *testing.T, r http.Handler, token string) map[string]any {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/settings", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("读取设置失败：%s", e.Message)
	}
	var data struct {
		Settings map[string]any `json:"settings"`
	}
	decodeInto(t, e, &data)
	return data.Settings
}

func putSettings(t *testing.T, r http.Handler, token string, settings map[string]any) {
	t.Helper()
	rec := doJSON(t, r, http.MethodPut, "/api/v1/admin/settings", token, settings)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("保存设置失败：%s", e.Message)
	}
}
