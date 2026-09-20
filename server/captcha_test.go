package main

// 图形验证码（#86/#11）：取题、校验、单次有效

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"myblog/server/handler"
)

func TestCaptchaFlow(t *testing.T) {
	r := newTestApp(t)

	// 取题：返回 id 与 SVG
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/auth/captcha", "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("取验证码失败：%s", e.Message)
	}
	var data struct {
		CaptchaID string `json:"captchaId"`
		Image     string `json:"image"`
	}
	decodeInto(t, e, &data)
	if data.CaptchaID == "" || !strings.Contains(data.Image, "<svg") {
		t.Fatalf("验证码响应异常：%+v", data)
	}

	// 答案可从同进程存储读取（测试通道），读取不消费
	answer := handler.GetCaptchaAnswer(data.CaptchaID)
	if answer == "" {
		t.Fatal("答案应可从同进程存储读取（测试通道）")
	}

	// 错误验证码 → 10001（提示为「不正确」而非「已过期」），
	// 且不消耗密码失败计数；该题随即销毁（单次有效）
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/auth/login", "",
		map[string]any{"username": "admin", "password": "admin123", "captchaId": data.CaptchaID, "captchaCode": "0000"})
	if e = decode(t, rec); e.Code != 10001 {
		t.Fatalf("错误验证码应 10001，got %d", e.Code)
	}
	if !strings.Contains(e.Message, "不正确") {
		t.Fatalf("答案不符时应提示「不正确」，got %q", e.Message)
	}
	if handler.GetCaptchaAnswer(data.CaptchaID) != "" {
		t.Fatal("错误尝试后验证码应被销毁")
	}

	// 重新取题走正确流程
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/auth/captcha", "", nil)
	decodeInto(t, decode(t, rec), &data)
	answer = handler.GetCaptchaAnswer(data.CaptchaID)
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/auth/login", "",
		map[string]any{"username": "admin", "password": "admin123", "captchaId": data.CaptchaID, "captchaCode": answer})
	if e = decode(t, rec); e.Code != 0 {
		t.Fatalf("正确验证码应登录成功：%s", e.Message)
	}

	// 登录成功后验证码已被消费：再次使用 → 10001，且提示为「已过期」
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/auth/login", "",
		map[string]any{"username": "admin", "password": "admin123", "captchaId": data.CaptchaID, "captchaCode": answer})
	if e = decode(t, rec); e.Code != 10001 {
		t.Fatalf("验证码应单次有效，got %d", e.Code)
	}
	if !strings.Contains(e.Message, "已过期") {
		t.Fatalf("题目已被消费时应提示「已过期」，got %q", e.Message)
	}
}

// TestCaptchaAnswerCaseInsensitive 字符集含大小写字母，用户大小写不一致时应放过，
// 且忽略首尾空格（移动端输入法常带空格）。
func TestCaptchaAnswerCaseInsensitive(t *testing.T) {
	r := newTestApp(t)

	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/auth/captcha", "", nil)
	e := decode(t, rec)
	var d struct {
		CaptchaID string `json:"captchaId"`
	}
	decodeInto(t, e, &d)

	answer := handler.GetCaptchaAnswer(d.CaptchaID)
	if answer == "" {
		t.Fatal("应能读到答案")
	}
	// 取反大小写 + 包一层空格
	swap := strings.Map(func(ch rune) rune {
		if ch >= 'a' && ch <= 'z' {
			return ch - 32
		}
		if ch >= 'A' && ch <= 'Z' {
			return ch + 32
		}
		return ch
	}, answer)

	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/auth/login", "",
		map[string]any{"username": "admin", "password": "admin123",
			"captchaId": d.CaptchaID, "captchaCode": "  " + swap + " "})
	if e = decode(t, rec); e.Code != 0 {
		t.Fatalf("大小写/空格差异应被容忍，got code=%d msg=%s", e.Code, e.Message)
	}
}

// TestCaptchaResponseNoStore 验证码响应必须禁止缓存，
// 否则中间层缓存会让客户端反复拿到已消费的 id（症状：一直刷新一直过期）。
func TestCaptchaResponseNoStore(t *testing.T) {
	r := newTestApp(t)
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/auth/captcha", "", nil)
	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "no-store") {
		t.Fatalf("验证码响应应带 no-store，got %q", cc)
	}
}

// TestCaptchaConcurrentIssueDistinct 并发取题必须产生互不相同的 id，
// 且每张题目的答案独立（防止 store 竞态导致串号）。
func TestCaptchaConcurrentIssueDistinct(t *testing.T) {
	r := newTestApp(t)

	const n = 40
	ids := make([]string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/auth/captcha", "", nil)
			var d struct {
				CaptchaID string `json:"captchaId"`
			}
			decodeInto(t, decode(t, rec), &d)
			ids[i] = d.CaptchaID
		}(i)
	}
	wg.Wait()

	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" {
			t.Fatal("并发取题出现空 id")
		}
		if seen[id] {
			t.Fatalf("并发取题出现重复 id：%s", id)
		}
		seen[id] = true
		if handler.GetCaptchaAnswer(id) == "" {
			t.Fatalf("id %s 应有独立答案", id)
		}
	}
}
