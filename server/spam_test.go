package main

// 评论 Spam 防护 + 黑名单 + 批量操作 + 通知中心（契约 #5/#31/#69-75）

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"myblog/server/middleware"
	"myblog/server/model"
)

func postComment(t *testing.T, r http.Handler, post model.AdminPostItemDTO, content, email string) (int, int8) {
	t.Helper()
	middleware.ResetCommentRateLimit() // 测试内绕过 60s 评论限流
	rec := doJSON(t, r, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "", map[string]any{
		"nickname": "访客", "email": email, "content": content,
	})
	e := decode(t, rec)
	if e.Code != 0 {
		return e.Code, -1
	}
	var data struct {
		Comment struct {
			ID uint `json:"id"`
		} `json:"comment"`
	}
	decodeInto(t, e, &data)
	var cm model.Comment
	if err := model.DB.First(&cm, data.Comment.ID).Error; err != nil {
		t.Fatalf("读取评论失败：%v", err)
	}
	return 0, cm.Status
}

func TestCommentSpamRules(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	post := createPost(t, r, token, map[string]any{"title": "spam 文", "status": 1})

	// 1. 正常评论 → 待审
	if code, st := postComment(t, r, post, "正常评论内容", "a@x.com"); code != 0 || st != model.CommentPending {
		t.Fatalf("正常评论应待审，code=%d status=%d", code, st)
	}

	// 2. 相同内容重复提交 → 垃圾
	if _, st := postComment(t, r, post, "正常评论内容", "b@x.com"); st != model.CommentSpam {
		t.Fatalf("重复内容应判垃圾，status=%d", st)
	}

	// 3. 链接过多（≥3）→ 垃圾
	spammy := "看这里 http://a.com http://b.com http://c.com"
	if _, st := postComment(t, r, post, spammy, "c@x.com"); st != model.CommentSpam {
		t.Fatalf("链接过多应判垃圾，status=%d", st)
	}

	// 4. 关键词黑名单
	model.DB.Create(&model.CommentBlacklist{Type: "keyword", Value: "广告"})
	if _, st := postComment(t, r, post, "这是Ads 广告 内容", "d@x.com"); st != model.CommentSpam {
		t.Fatalf("关键词黑名单应判垃圾，status=%d", st)
	}

	// 5. 邮箱黑名单
	model.DB.Create(&model.CommentBlacklist{Type: "email", Value: "bad@spam.io"})
	if _, st := postComment(t, r, post, "别的评论", strings.ToUpper("BAD@SPAM.IO")); st != model.CommentSpam {
		t.Fatalf("邮箱黑名单应判垃圾，status=%d", st)
	}
}

func TestCommentBlacklistAndBatch(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	post := createPost(t, r, token, map[string]any{"title": "批量文", "status": 1})

	// 黑名单 CRUD
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/comment-blacklist", token,
		map[string]any{"type": "ip", "value": "1.2.3.4"})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("建黑名单失败：%s", e.Message)
	}
	// 重复 → 10001
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/comment-blacklist", token,
		map[string]any{"type": "ip", "value": "1.2.3.4"})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("重复黑名单应 10001，got %d", e.Code)
	}
	// 非法 type → 10001
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/comment-blacklist", token,
		map[string]any{"type": "ua", "value": "x"})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("非法 type 应 10001，got %d", e.Code)
	}
	// 列表
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/comment-blacklist?type=ip", token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("黑名单列表失败：%s", e.Message)
	}

	// 三条评论：2 待审 + 1 通过
	var ids []uint
	for i := 0; i < 3; i++ {
		middleware.ResetCommentRateLimit() // 绕过 60s 评论限流
		rec = doJSON(t, r, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "", map[string]any{
			"nickname": fmt.Sprintf("访客%d", i), "email": "batch@x.com",
			"content": fmt.Sprintf("待审内容 %d", i),
		})
		e := decode(t, rec)
		if e.Code != 0 {
			t.Fatalf("评论失败：%s", e.Message)
		}
		var data struct {
			Comment struct {
				ID uint `json:"id"`
			} `json:"comment"`
		}
		decodeInto(t, e, &data)
		ids = append(ids, data.Comment.ID)
	}
	// 通过第一条
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/comments/%d/status", ids[0]), token,
		map[string]any{"status": 1})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("通过失败：%s", e.Message)
	}

	// 批量标垃圾 ids[1..2]
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/comments/batch", token,
		map[string]any{"action": "spam", "ids": []uint{ids[1], ids[2]}})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("批量失败：%s", e.Message)
	}
	var spamN int64
	model.DB.Model(&model.Comment{}).Where("status = ?", model.CommentSpam).Count(&spamN)
	if spamN != 2 {
		t.Fatalf("批量标垃圾后应有 2 条，got %d", spamN)
	}

	// 批量删除（含子回复级联）
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/comments/batch", token,
		map[string]any{"action": "delete", "ids": ids[:2]})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("批量删除失败：%s", e.Message)
	}
	var left int64
	model.DB.Model(&model.Comment{}).Where("post_id = ?", post.ID).Count(&left)
	if left != 1 {
		t.Fatalf("批量删除后该文章应剩 1 条，got %d", left)
	}

	// 回收站流转：单条 status=4 再恢复 0
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/comments/%d/status", ids[2]), token,
		map[string]any{"status": 4})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("移入回收站失败：%s", e.Message)
	}
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/comments?status=4", token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("回收站筛选失败：%s", e.Message)
	}
}

func TestNotifications(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	post := createPost(t, r, token, map[string]any{"title": "通知文", "status": 1})

	// 一条待审评论 → comment_pending 通知
	rec := doJSON(t, r, http.MethodPost, "/api/v1/posts/"+post.Slug+"/comments", "", map[string]any{
		"nickname": "通知访客", "email": "n@x.com", "content": "通知内容",
	})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("评论失败：%s", e.Message)
	}

	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/notifications", token, nil)
	e := decode(t, rec)
	var data struct {
		List []struct {
			Type  string `json:"type"`
			Read  bool   `json:"read"`
			Title string `json:"title"`
		} `json:"list"`
		UnreadCount int64 `json:"unreadCount"`
	}
	decodeInto(t, e, &data)
	if len(data.List) == 0 || data.List[0].Type != "comment_pending" {
		t.Fatalf("应产生 comment_pending 通知，got %+v", data.List)
	}
	if data.UnreadCount != 1 {
		t.Fatalf("未读数应为 1，got %d", data.UnreadCount)
	}

	// 全部已读
	rec = doJSON(t, r, http.MethodPut, "/api/v1/admin/notifications/read-all", token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("read-all 失败：%s", e.Message)
	}
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/notifications", token, nil)
	decodeInto(t, decode(t, rec), &data)
	if data.UnreadCount != 0 || data.List[0].Read != true {
		t.Fatalf("已读后 unread 应为 0，got %+v", data)
	}

	// 单条已读端点存在且 200
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/notifications/%d/read", 1), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("单条已读失败：code=%d", e.Code)
	}
}
