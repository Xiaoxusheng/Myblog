package main

// 防护体系集成测试（契约「安全防护」）：WAF 拦截/日志模式、全局限流、
// 违规计点自动封禁、白名单、手动封禁管理接口、封禁持久化（重启恢复）、
// 登录爆破升级封禁。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"myblog/server/config"
	"myblog/server/handler"
	"myblog/server/middleware"
	"myblog/server/model"
	"myblog/server/router"

	"github.com/gin-gonic/gin"
)

const (
	defenseIP   = "203.0.113.10" // 测试用攻击方 IP
	defenseIP2  = "198.51.100.7" // 测试用另一攻击方 IP
	defenseUA   = "defense-test-agent"
	testAdminIP = "192.0.2.1" // httptest.NewRequest 默认来源 IP
)

// doReq 发起带来源 IP 与 User-Agent 的请求（防护按 UA/IP 判定，必须可控）
func doReq(t *testing.T, r http.Handler, method, path, ip, ua string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = ip + ":12345"
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// doJSONFrom 同 doJSON，但可指定来源 IP（登录爆破用例需要）；带测试 UA 以通过 WAF
func doJSONFrom(t *testing.T, r http.Handler, method, path, ip string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Buffer
	if payload != nil {
		body = new(bytes.Buffer)
		if err := json.NewEncoder(body).Encode(payload); err != nil {
			t.Fatalf("编码请求体失败：%v", err)
		}
	} else {
		body = new(bytes.Buffer)
	}
	req := httptest.NewRequest(method, path, body)
	req.RemoteAddr = ip + ":12345"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", defenseUA)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// defenseApp 构建开启防护的测试应用：默认 WAF block，限流/自动封禁由用例按需开启。
// 管理端请求来源（httptest 默认 IP）默认加白，避免空 UA 的 doJSON 被 WAF 干扰；
// doJSONFrom 显式带 UA，攻击方 IP 不在白名单内。
func defenseApp(t *testing.T, tune func(*config.Config)) *gin.Engine {
	t.Helper()
	return newTestAppCfg(t, func(cfg *config.Config) {
		cfg.WAFMode = "block"
		cfg.RateLimitPerMin = 0
		cfg.BanThreshold = 0
		cfg.BanWindow = time.Minute
		cfg.BanDuration = time.Minute
		cfg.BanWhitelist = []string{testAdminIP}
		if tune != nil {
			tune(cfg)
		}
	})
}

func TestDefenseWAFBlocksMaliciousRequests(t *testing.T) {
	r := defenseApp(t, nil)

	cases := []struct {
		name string
		path string
		ua   string
	}{
		{"攻击工具UA", "/api/v1/site", "sqlmap/1.7.11#stable"},
		{"CMS探测", "/wp-login.php", defenseUA},
		{"隐藏env文件", "/.env", defenseUA},
		{"脚本探测", "/eval-stdin.php", defenseUA},
		{"git目录探测", "/.git/config", defenseUA},
		{"查询串注入", "/api/v1/site?id=1%20union%20select%20null", defenseUA},
		{"查询串穿越", "/api/v1/site?next=../../etc/passwd", defenseUA},
		{"空UA", "/api/v1/site", ""},
	}
	for _, tc := range cases {
		rec := doReq(t, r, http.MethodGet, tc.path, defenseIP, tc.ua)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s：期望 403，实际 %d（body: %s）", tc.name, rec.Code, rec.Body.String())
		}
		var body struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Code != 10003 {
			t.Fatalf("%s：期望错误码 10003，实际 %+v err=%v", tc.name, body, err)
		}
	}

	// 正常请求不受影响
	if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusOK {
		t.Fatalf("正常请求被误拦截：%d", rec.Code)
	}
	// 搜索接口豁免注入特征检查（技术博客检索 SQL 关键词是正常需求）
	if rec := doReq(t, r, http.MethodGet, "/api/v1/posts?keyword=union%20select", defenseIP, defenseUA); rec.Code != http.StatusOK {
		t.Fatalf("搜索关键词被误拦截：%d", rec.Code)
	}
	// 搜索接口的目录穿越仍然拦截
	if rec := doReq(t, r, http.MethodGet, "/api/v1/posts?keyword=../../etc", defenseIP, defenseUA); rec.Code != http.StatusForbidden {
		t.Fatalf("搜索目录穿越未被拦截：%d", rec.Code)
	}
}

func TestDefenseWAFLogModeOnlyRecords(t *testing.T) {
	r := defenseApp(t, func(cfg *config.Config) { cfg.WAFMode = "log" })

	// log 模式不拦截：探测路径走正常路由（404 = 未命中路由）
	if rec := doReq(t, r, http.MethodGet, "/wp-login.php", defenseIP, defenseUA); rec.Code != http.StatusNotFound {
		t.Fatalf("log 模式不应拦截，期望 404，实际 %d", rec.Code)
	}
	// 正常接口照常可用（含空 UA）
	if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, ""); rec.Code != http.StatusOK {
		t.Fatalf("log 模式下空 UA 请求被误拦截：%d", rec.Code)
	}
	// 事件已记录
	events := middleware.CurrentGuard().ListEvents(10)
	if len(events) == 0 || events[0].Kind != "waf" {
		t.Fatalf("log 模式应记录 waf 事件，实际 %+v", events)
	}
}

func TestDefenseRateLimitAndAutoBan(t *testing.T) {
	r := defenseApp(t, func(cfg *config.Config) {
		cfg.RateLimitPerMin = 3
		cfg.BanThreshold = 3
	})

	// 前 3 次放行
	for i := 0; i < 3; i++ {
		if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusOK {
			t.Fatalf("第 %d 次请求应放行，实际 %d", i+1, rec.Code)
		}
	}
	// 第 4-6 次超限 429（各计 1 点），第 3 点触发自动封禁
	for i := 0; i < 3; i++ {
		rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("第 %d 次超限请求应 429，实际 %d", i+4, rec.Code)
		}
		if rec.Header().Get("Retry-After") == "" {
			t.Fatal("429 响应缺少 Retry-After")
		}
	}
	// 已自动封禁：直接 403
	if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusForbidden {
		t.Fatalf("封禁后应 403，实际 %d", rec.Code)
	}
	// 封禁已持久化
	var n int64
	model.DB.Model(&model.BannedIP{}).Where("ip = ?", defenseIP).Count(&n)
	if n != 1 {
		t.Fatalf("自动封禁应落库，banned_ips 命中 %d 条", n)
	}

	// 管理端可见（source=auto），解除后恢复（仍超限 → 429 而非 403，证明未再封禁）
	token := loginToken(t, r)
	e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/security/bans", token, nil))
	if e.Code != 0 {
		t.Fatalf("查询封禁列表失败：%s", e.Message)
	}
	var listData struct {
		List []struct {
			IP     string `json:"ip"`
			Source string `json:"source"`
		} `json:"list"`
	}
	decodeInto(t, e, &listData)
	if len(listData.List) != 1 || listData.List[0].IP != defenseIP || listData.List[0].Source != "auto" {
		t.Fatalf("封禁列表不符：%+v", listData.List)
	}

	e = decode(t, doJSON(t, r, http.MethodDelete, "/api/v1/admin/security/bans/"+defenseIP, token, nil))
	if e.Code != 0 {
		t.Fatalf("解除封禁失败：%s", e.Message)
	}
	if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("解封后应回到 429（限流仍生效），实际 %d", rec.Code)
	}
}

func TestDefenseWhitelistBypassesGuard(t *testing.T) {
	r := defenseApp(t, func(cfg *config.Config) {
		cfg.RateLimitPerMin = 1
		cfg.BanThreshold = 1
		cfg.BanWhitelist = []string{"203.0.113.99"}
	})

	// 白名单 IP：超限也不拦
	for i := 0; i < 5; i++ {
		if rec := doReq(t, r, http.MethodGet, "/api/v1/site", "203.0.113.99", defenseUA); rec.Code != http.StatusOK {
			t.Fatalf("白名单 IP 第 %d 次请求应放行，实际 %d", i+1, rec.Code)
		}
	}
	// 非白名单 IP：第 2 次触发 429 计 1 点，即达阈值（threshold=1）→ 封禁 → 403
	if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusOK {
		t.Fatalf("首次请求应放行，实际 %d", rec.Code)
	}
	if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("第 2 次请求应 429，实际 %d", rec.Code)
	}
	if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusForbidden {
		t.Fatalf("计点达阈值后应 403，实际 %d", rec.Code)
	}
}

func TestDefenseManualBanAPI(t *testing.T) {
	r := defenseApp(t, nil)
	token := loginToken(t, r)

	// 非法 IP → 10001
	e := decode(t, doJSON(t, r, http.MethodPost, "/api/v1/admin/security/bans", token,
		map[string]any{"ip": "999.9.9.9"}))
	if e.Code != 10001 {
		t.Fatalf("非法 IP 应 10001，实际 %d", e.Code)
	}
	// 封禁自己 → 10001
	e = decode(t, doJSON(t, r, http.MethodPost, "/api/v1/admin/security/bans", token,
		map[string]any{"ip": testAdminIP}))
	if e.Code != 10001 {
		t.Fatalf("封禁当前登录 IP 应 10001，实际 %d", e.Code)
	}

	// 手动封禁 60 分钟
	e = decode(t, doJSON(t, r, http.MethodPost, "/api/v1/admin/security/bans", token,
		map[string]any{"ip": defenseIP, "durationMinutes": 60, "reason": "滥用爬虫"}))
	if e.Code != 0 {
		t.Fatalf("手动封禁失败：%s", e.Message)
	}
	if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusForbidden {
		t.Fatalf("手动封禁后应 403，实际 %d", rec.Code)
	}

	// 解除封禁 → 恢复；重复解除 → 10004
	e = decode(t, doJSON(t, r, http.MethodDelete, "/api/v1/admin/security/bans/"+defenseIP, token, nil))
	if e.Code != 0 {
		t.Fatalf("解除封禁失败：%s", e.Message)
	}
	if rec := doReq(t, r, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusOK {
		t.Fatalf("解封后应恢复 200，实际 %d", rec.Code)
	}
	e = decode(t, doJSON(t, r, http.MethodDelete, "/api/v1/admin/security/bans/"+defenseIP, token, nil))
	if e.Code != 10004 {
		t.Fatalf("重复解除应 10004，实际 %d", e.Code)
	}

	// 操作已写审计日志
	e = decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs?action=security.", token, nil))
	var auditData struct {
		Total int64 `json:"total"`
	}
	decodeInto(t, e, &auditData)
	if auditData.Total < 2 {
		t.Fatalf("security.ban/unban 应写入审计日志，实际 %d 条", auditData.Total)
	}
}

func TestDefenseBanPersistsAcrossRestart(t *testing.T) {
	var captured *config.Config
	r := defenseApp(t, func(cfg *config.Config) { captured = cfg })
	token := loginToken(t, r)

	// 永久封禁（durationMinutes 缺省 = 0）
	e := decode(t, doJSON(t, r, http.MethodPost, "/api/v1/admin/security/bans", token,
		map[string]any{"ip": defenseIP2, "reason": "永久封禁测试"}))
	if e.Code != 0 {
		t.Fatalf("永久封禁失败：%s", e.Message)
	}
	// 临时封禁（1ms，等待过期后重启不应加载）
	middleware.CurrentGuard().Ban(defenseIP, "临时封禁", time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	// 同一数据库重新装配引擎+Guard，模拟进程重启
	r2 := router.Setup(captured)
	bans := middleware.CurrentGuard().ListBans()
	if len(bans) != 1 || bans[0].IP != defenseIP2 {
		t.Fatalf("重启后应仅加载未过期的永久封禁，实际 %+v", bans)
	}
	if rec := doReq(t, r2, http.MethodGet, "/api/v1/site", defenseIP2, defenseUA); rec.Code != http.StatusForbidden {
		t.Fatalf("重启后封禁应仍生效，实际 %d", rec.Code)
	}
	if rec := doReq(t, r2, http.MethodGet, "/api/v1/site", defenseIP, defenseUA); rec.Code != http.StatusOK {
		t.Fatalf("已过期的临时封禁不应生效，实际 %d", rec.Code)
	}
}

func TestDefenseLoginBruteforceEscalatesToBan(t *testing.T) {
	r := defenseApp(t, func(cfg *config.Config) {
		cfg.BanThreshold = 5
		cfg.BanWindow = time.Minute
		cfg.BanDuration = time.Minute
	})

	// 5 次错误密码 → 触发登录锁定（20001）
	for i := 0; i < 5; i++ {
		e := decode(t, attemptLoginFrom(t, r, defenseIP, "admin", "wrong-password"))
		if e.Code != 20001 {
			t.Fatalf("第 %d 次错误登录应 20001，实际 %d", i+1, e.Code)
		}
	}
	// 锁定期内继续尝试：每次计 1 点，5 点触发自动封禁
	for i := 0; i < 5; i++ {
		e := decode(t, attemptLoginFrom(t, r, defenseIP, "admin", "wrong-password"))
		if e.Code != 20003 {
			t.Fatalf("锁定期间第 %d 次尝试应 20003，实际 %d", i+1, e.Code)
		}
	}
	// 已自动封禁：403 在引擎层短路，请求不再进入登录流程
	rec := doJSONFrom(t, r, http.MethodPost, "/api/v1/admin/auth/login", defenseIP,
		map[string]any{"username": "admin", "password": "x"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("爆破 IP 应被自动封禁 403，实际 %d", rec.Code)
	}
	// 封禁记录 source=auto
	token := loginToken(t, r)
	e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/security/bans", token, nil))
	var listData struct {
		List []struct {
			IP     string `json:"ip"`
			Source string `json:"source"`
		} `json:"list"`
	}
	decodeInto(t, e, &listData)
	if len(listData.List) != 1 || listData.List[0].IP != defenseIP || listData.List[0].Source != "auto" {
		t.Fatalf("自动封禁记录不符：%+v", listData.List)
	}
}

func TestDefenseEventsAPI(t *testing.T) {
	r := defenseApp(t, nil)

	doReq(t, r, http.MethodGet, "/wp-login.php", defenseIP, defenseUA) // 触发一条 waf 事件
	token := loginToken(t, r)

	e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/security/events?limit=50", token, nil))
	if e.Code != 0 {
		t.Fatalf("查询防护事件失败：%s", e.Message)
	}
	var data struct {
		List []middleware.DefenseEvent `json:"list"`
	}
	decodeInto(t, e, &data)
	if len(data.List) == 0 {
		t.Fatal("应至少有 1 条防护事件")
	}
	latest := data.List[0]
	if latest.Kind != "waf" || latest.IP != defenseIP || latest.Detail == "" {
		t.Fatalf("最新事件不符：%+v", latest)
	}
}

// attemptLoginFrom 同 attemptLogin，但可指定来源 IP
func attemptLoginFrom(t *testing.T, r http.Handler, ip, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	capRec := doJSONFrom(t, r, http.MethodGet, "/api/v1/admin/auth/captcha", ip, nil)
	capEnv := decode(t, capRec)
	var capData struct {
		CaptchaID string `json:"captchaId"`
	}
	decodeInto(t, capEnv, &capData)
	captchaCode := handler.GetCaptchaAnswer(capData.CaptchaID)
	return doJSONFrom(t, r, http.MethodPost, "/api/v1/admin/auth/login", ip,
		map[string]any{
			"username": username, "password": password,
			"captchaId": capData.CaptchaID, "captchaCode": captchaCode,
		})
}
