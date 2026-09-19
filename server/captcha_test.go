package main

// 图形验证码（#86/#11）：取题、校验、单次有效

import (
	"net/http"
	"strings"
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

	// 错误验证码 → 10001，且不消耗密码失败计数；该题随即销毁（单次有效）
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/auth/login", "",
		map[string]any{"username": "admin", "password": "admin123", "captchaId": data.CaptchaID, "captchaCode": "0000"})
	if e = decode(t, rec); e.Code != 10001 {
		t.Fatalf("错误验证码应 10001，got %d", e.Code)
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

	// 登录成功后验证码已被消费：再次使用 → 10001
	rec = doJSON(t, r, http.MethodPost, "/api/v1/admin/auth/login", "",
		map[string]any{"username": "admin", "password": "admin123", "captchaId": data.CaptchaID, "captchaCode": answer})
	if e = decode(t, rec); e.Code != 10001 {
		t.Fatalf("验证码应单次有效，got %d", e.Code)
	}
}
