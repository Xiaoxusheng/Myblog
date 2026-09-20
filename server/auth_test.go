package main

import (
	"net/http"
	"strings"
	"testing"

	"myblog/server/handler"
)

// 登录成功：code 0，token 非空，user 为契约 User 对象
func TestLoginSuccess(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)
	if token == "" {
		t.Fatal("token 为空")
	}
}

// 登录失败：密码错误 → HTTP 200 + code 20001
func TestLoginWrongPassword(t *testing.T) {
	app := newTestApp(t)
	rec := attemptLogin(t, app, "admin", "wrong-password")
	if rec.Code != http.StatusOK {
		t.Fatalf("业务错误应返回 HTTP 200，实际 %d", rec.Code)
	}
	e := decode(t, rec)
	if e.Code != 20001 {
		t.Fatalf("期望错误码 20001，实际 %d（%s）", e.Code, e.Message)
	}
}

// 关键语义：验证码正确但密码错误时，错误码必须是 20001 而非 10001。
// 前端依赖这个区分——10001 才刷新验证码，20001 不该连坐刷新，
// 否则用户每改一次密码都要重看验证码，会被误解成「验证码一直过期」。
func TestLoginWrongPasswordKeepsCaptchaSemantics(t *testing.T) {
	app := newTestApp(t)

	// attemptLogin 内部会取一题并用正确答案提交，因此这里得到的是「验证码已通过」的响应
	rec := attemptLogin(t, app, "admin", "wrong-password")
	e := decode(t, rec)
	if e.Code != 20001 {
		t.Fatalf("验证码正确 + 密码错误 应返回 20001，实际 %d（%s）", e.Code, e.Message)
	}
	if strings.Contains(e.Message, "验证码") {
		t.Fatalf("20001 的提示不应涉及验证码，实际 %q", e.Message)
	}

	// 反向确认：同一题被消费后重用 → 10001（这才是「已过期」的来源）
	capRec := doJSON(t, app, http.MethodGet, "/api/v1/admin/auth/captcha", "", nil)
	capEnv := decode(t, capRec)
	var capData struct {
		CaptchaID string `json:"captchaId"`
	}
	decodeInto(t, capEnv, &capData)
	answer := handler.GetCaptchaAnswer(capData.CaptchaID)

	payload := map[string]any{
		"username": "admin", "password": "admin123",
		"captchaId": capData.CaptchaID, "captchaCode": answer,
	}
	// 第一次：验证码正确 + 密码正确 → 成功并消费该题
	if e := decode(t, doJSON(t, app, http.MethodPost, "/api/v1/admin/auth/login", "", payload)); e.Code != 0 {
		t.Fatalf("首次登录应成功：%s", e.Message)
	}
	// 第二次用同一题 → 10001，提示「已过期」
	e2 := decode(t, doJSON(t, app, http.MethodPost, "/api/v1/admin/auth/login", "", payload))
	if e2.Code != 10001 {
		t.Fatalf("复用已消费的验证码应 10001，实际 %d", e2.Code)
	}
	if !strings.Contains(e2.Message, "已过期") {
		t.Fatalf("复用已消费的验证码应提示「已过期」，实际 %q", e2.Message)
	}
}

// 未带 token 访问 /admin/* → HTTP 401 + code 10002
func TestAdminRequiresAuth(t *testing.T) {
	app := newTestApp(t)
	for _, path := range []string{
		"/api/v1/admin/posts",
		"/api/v1/admin/stats",
		"/api/v1/admin/comments",
		"/api/v1/admin/settings",
		"/api/v1/admin/uploads",
	} {
		rec := doJSON(t, app, http.MethodGet, path, "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s 未带 token 应 401，实际 %d", path, rec.Code)
		}
		e := decode(t, rec)
		if e.Code != 10002 {
			t.Fatalf("%s 期望 code 10002，实际 %d", path, e.Code)
		}
	}

	// 伪造 token 同样 401
	rec := doJSON(t, app, http.MethodGet, "/api/v1/admin/posts", "forged-token", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("伪造 token 应 401，实际 %d", rec.Code)
	}
}

// me / profile：携带合法 token 正常返回
func TestMeAndProfile(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	rec := doJSON(t, app, http.MethodGet, "/api/v1/admin/auth/me", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("me 失败：%s", e.Message)
	}
	var me struct {
		User struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"user"`
	}
	decodeInto(t, e, &me)
	if me.User.Username != "admin" || me.User.ID == 0 {
		t.Fatalf("me 返回异常：%+v", me.User)
	}
	if me.User.Password != "" {
		t.Fatal("密码字段不应外泄")
	}

	// 更新资料
	rec = doJSON(t, app, http.MethodPut, "/api/v1/admin/auth/profile", token,
		map[string]any{"nickname": "站长", "email": "owner@example.com", "avatar": ""})
	e = decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("更新资料失败：%s", e.Message)
	}
	var prof struct {
		User struct {
			Nickname string `json:"nickname"`
		} `json:"user"`
	}
	decodeInto(t, e, &prof)
	if prof.User.Nickname != "站长" {
		t.Fatalf("昵称未更新：%s", prof.User.Nickname)
	}
}

// 修改密码：旧密码错误 20001；改密后新密码可登录、旧密码失效
func TestChangePassword(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	// 旧密码错误
	rec := doJSON(t, app, http.MethodPut, "/api/v1/admin/auth/password", token,
		map[string]any{"oldPassword": "bad-old", "newPassword": "newpass123"})
	if e := decode(t, rec); e.Code != 20001 {
		t.Fatalf("旧密码错误应 20001，实际 %d", e.Code)
	}

	// 新密码太短
	rec = doJSON(t, app, http.MethodPut, "/api/v1/admin/auth/password", token,
		map[string]any{"oldPassword": "admin123", "newPassword": "123"})
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("新密码过短应 10001，实际 %d", e.Code)
	}

	// 正常修改
	rec = doJSON(t, app, http.MethodPut, "/api/v1/admin/auth/password", token,
		map[string]any{"oldPassword": "admin123", "newPassword": "newpass123"})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("修改密码失败：%s", e.Message)
	}

	// 旧密码登录失败
	rec = attemptLogin(t, app, "admin", "admin123")
	if e := decode(t, rec); e.Code != 20001 {
		t.Fatalf("旧密码登录应 20001，实际 %d", e.Code)
	}
	// 新密码登录成功
	rec = attemptLogin(t, app, "admin", "newpass123")
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("新密码登录失败：%s", e.Message)
	}
}

// 仪表盘统计
func TestStats(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	// 种子：3 已发布 + 1 草稿；评论 3 条（张三、管理员回复、李四）
	createPost(t, app, token, map[string]any{"title": "统计用文章", "status": 1})

	rec := doJSON(t, app, http.MethodGet, "/api/v1/admin/stats", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("stats 失败：%s", e.Message)
	}
	var data struct {
		PostCount           int64 `json:"postCount"`
		DraftCount          int64 `json:"draftCount"`
		CommentCount        int64 `json:"commentCount"`
		PendingCommentCount int64 `json:"pendingCommentCount"`
		ViewCount           int64 `json:"viewCount"`
		LikeCount           int64 `json:"likeCount"`
		LinkCount           int64 `json:"linkCount"`
		Trend               []struct {
			Date     string `json:"date"`
			Posts    int64  `json:"posts"`
			Comments int64  `json:"comments"`
		} `json:"trend"`
		RecentComments []struct {
			ID        uint   `json:"id"`
			PostTitle string `json:"postTitle"`
			Nickname  string `json:"nickname"`
		} `json:"recentComments"`
	}
	decodeInto(t, e, &data)

	if data.PostCount != 5 || data.DraftCount != 1 {
		t.Fatalf("文章统计错误：postCount=%d draftCount=%d", data.PostCount, data.DraftCount)
	}
	if data.CommentCount != 3 || data.PendingCommentCount != 0 {
		t.Fatalf("评论统计错误：commentCount=%d pending=%d", data.CommentCount, data.PendingCommentCount)
	}
	if data.LinkCount != 3 {
		t.Fatalf("友链数错误：%d", data.LinkCount)
	}
	if len(data.Trend) != 7 {
		t.Fatalf("趋势应为近 7 天，实际 %d", len(data.Trend))
	}
	today := data.Trend[len(data.Trend)-1]
	if today.Posts == 0 {
		t.Fatal("今日发布数应为正（种子+新建均在本周）")
	}
	if len(data.RecentComments) == 0 || data.RecentComments[0].PostTitle == "" {
		t.Fatal("最近评论缺少文章标题")
	}
}
