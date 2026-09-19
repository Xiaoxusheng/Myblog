package main

import (
	"encoding/hex"
	"net/http"
	"strings"
	"testing"

	"myblog/server/config"
	"myblog/server/middleware"
	"myblog/server/model"
)

// piiRawRow 原始行（绕过 GORM serializer，直读库中真实存储值）
type piiRawRow struct {
	Email string
	IP    string
}

// createGuestComment 通过公开接口在种子文章下发一条游客评论，返回评论 id
func createGuestComment(t *testing.T, app http.Handler, email string) uint {
	t.Helper()
	rec := doJSON(t, app, http.MethodPost, "/api/v1/posts/gin-blog-backend/comments", "", map[string]any{
		"nickname": "测试游客", "email": email, "content": "加密测试评论",
	})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("发表评论失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Comment struct {
			ID uint `json:"id"`
		} `json:"comment"`
	}
	decodeInto(t, e, &data)
	if data.Comment.ID == 0 {
		t.Fatal("评论 id 为空")
	}
	return data.Comment.ID
}

func rawCommentPII(t *testing.T, id uint) piiRawRow {
	t.Helper()
	var row piiRawRow
	if err := model.DB.Raw("SELECT email, ip FROM comments WHERE id = ?", id).Scan(&row).Error; err != nil {
		t.Fatalf("读取原始评论行失败：%v", err)
	}
	return row
}

func rawAdminEmail(t *testing.T) string {
	t.Helper()
	var email string
	if err := model.DB.Raw("SELECT email FROM users WHERE username = ?", "admin").Scan(&email).Error; err != nil {
		t.Fatalf("读取原始用户邮箱失败：%v", err)
	}
	return email
}

// loginThenGetAdminComment 登录管理端并在评论列表中找到指定邮箱的评论
func loginThenGetAdminComment(t *testing.T, app http.Handler, email string) map[string]any {
	t.Helper()
	token := loginToken(t, app)
	rec := doJSON(t, app, http.MethodGet, "/api/v1/admin/comments", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("管理端评论列表失败：%s", e.Message)
	}
	var data struct {
		List []map[string]any `json:"list"`
	}
	decodeInto(t, e, &data)
	for _, cm := range data.List {
		if cm["email"] == email {
			return cm
		}
	}
	t.Fatalf("管理端列表未找到邮箱为 %s 的评论", email)
	return nil
}

// 启用加密后：评论邮箱/IP、管理员邮箱在库中均为密文，接口侧正常解密；
// profile 更新邮箱后同样落库为密文
func TestPIIEncryptedAtRest(t *testing.T) {
	app := newTestAppCfg(t, func(cfg *config.Config) {
		cfg.CryptoKey = testCryptoKeyHex
	})

	const visitorEmail = "secret-visitor@example.com"
	id := createGuestComment(t, app, visitorEmail)

	// 库中是密文：带 v1: 前缀，不含明文
	raw := rawCommentPII(t, id)
	if !strings.HasPrefix(raw.Email, "v1:") || strings.Contains(raw.Email, visitorEmail) {
		t.Fatalf("评论邮箱应为密文，实际 %q", raw.Email)
	}
	if !strings.HasPrefix(raw.IP, "v1:") {
		t.Fatalf("评论 IP 应为密文，实际 %q", raw.IP)
	}
	if !strings.HasPrefix(rawAdminEmail(t), "v1:") {
		t.Fatalf("管理员邮箱应为密文，实际 %q", rawAdminEmail(t))
	}

	// 管理端接口读到明文（解密路径）
	cm := loginThenGetAdminComment(t, app, visitorEmail)
	if ip, _ := cm["ip"].(string); ip == "" || strings.HasPrefix(ip, "v1:") {
		t.Fatalf("管理端评论 IP 应为解密后的明文，实际 %v", cm["ip"])
	}

	// 修改管理员资料后新邮箱仍为密文，me 接口可解密读出
	token := loginToken(t, app)
	const newEmail = "root-new@example.com"
	rec := doJSON(t, app, http.MethodPut, "/api/v1/admin/auth/profile", token,
		map[string]any{"nickname": "管理员", "email": newEmail, "avatar": ""})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("更新资料失败：%s", e.Message)
	}
	if !strings.HasPrefix(rawAdminEmail(t), "v1:") || strings.Contains(rawAdminEmail(t), newEmail) {
		t.Fatalf("更新后的管理员邮箱应为密文，实际 %q", rawAdminEmail(t))
	}
	rec = doJSON(t, app, http.MethodGet, "/api/v1/admin/auth/me", token, nil)
	e := decode(t, rec)
	var me struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	decodeInto(t, e, &me)
	if me.User.Email != newEmail {
		t.Fatalf("me 应返回解密后的邮箱 %s，实际 %s", newEmail, me.User.Email)
	}
}

// 存量明文数据：开启密钥后启动迁移加密，且能正常解密读取
func TestPIIMigrationEncryptsLegacyRows(t *testing.T) {
	app := newTestApp(t) // 未配置加密密钥 → 明文落库（模拟升级前数据）

	const legacyEmail = "legacy-visitor@example.com"
	id := createGuestComment(t, app, legacyEmail)
	if raw := rawCommentPII(t, id); strings.HasPrefix(raw.Email, "v1:") {
		t.Fatalf("迁移前评论邮箱应为明文，实际 %q", raw.Email)
	}

	// 注入密钥并执行迁移（幂等，可重复调用）
	key, err := hex.DecodeString(testCryptoKeyHex)
	if err != nil {
		t.Fatalf("解码测试密钥失败：%v", err)
	}
	if err := model.SetCryptoKey(key); err != nil {
		t.Fatalf("注入密钥失败：%v", err)
	}
	for i := 0; i < 2; i++ {
		if err := model.MigratePIIEncryption(model.DB); err != nil {
			t.Fatalf("第 %d 次迁移失败：%v", i+1, err)
		}
	}

	raw := rawCommentPII(t, id)
	if !strings.HasPrefix(raw.Email, "v1:") || strings.Contains(raw.Email, legacyEmail) {
		t.Fatalf("迁移后评论邮箱应为密文，实际 %q", raw.Email)
	}
	if !strings.HasPrefix(raw.IP, "v1:") {
		t.Fatalf("迁移后评论 IP 应为密文，实际 %q", raw.IP)
	}
	if !strings.HasPrefix(rawAdminEmail(t), "v1:") {
		t.Fatalf("迁移后管理员邮箱应为密文，实际 %q", rawAdminEmail(t))
	}

	// GORM 读取走解密路径，数据完好
	var cm model.Comment
	if err := model.DB.First(&cm, id).Error; err != nil {
		t.Fatalf("读取评论失败：%v", err)
	}
	if cm.Email != legacyEmail {
		t.Fatalf("解密后邮箱应还原为 %s，实际 %s", legacyEmail, cm.Email)
	}
	var user model.User
	if err := model.DB.Where("username = ?", "admin").First(&user).Error; err != nil {
		t.Fatalf("读取用户失败：%v", err)
	}
	if user.Email != "admin@example.com" {
		t.Fatalf("解密后管理员邮箱应还原，实际 %s", user.Email)
	}
}

// 登录防爆破：同 IP 连续失败 5 次后锁定，正确密码也被拒绝（20003）
func TestLoginRateLimit(t *testing.T) {
	app := newTestApp(t)

	for i := 0; i < 5; i++ {
		rec := attemptLogin(t, app, "admin", "wrong-password")
		if e := decode(t, rec); e.Code != 20001 {
			t.Fatalf("第 %d 次错误登录应返回 20001，实际 %d", i+1, e.Code)
		}
	}

	rec := attemptLogin(t, app, "admin", "admin123")
	e := decode(t, rec)
	if e.Code != 20003 {
		t.Fatalf("锁定后正确密码也应被拒绝（20003），实际 %d（%s）", e.Code, e.Message)
	}

	// 重置后（模拟 15 分钟窗口过期）恢复登录
	middleware.ResetLoginRateLimit()
	token := loginToken(t, app)
	if token == "" {
		t.Fatal("解锁后登录应成功")
	}
}
