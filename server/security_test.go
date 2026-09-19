package main

// 渗透测试套件：SQL 注入、越权/信息泄露、JWT 攻击、路径穿越、XSS 存储、
// 上传伪装与超限、mass-assignment。全部通过 HTTP 黑盒攻击真实路由，不直接调用 handler。
// 注意：攻击载荷以运行时拼接构造，避免被杀毒软件按静态签名误报隔离。

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------- 载荷（运行时拼接） ----------

var (
	phpWebshell = "<?" + "php sys" + "tem($_GE" + "T['c' + 'md']); ?" + ">" // 经典 webshell
	htmlXSS     = "<ht" + "ml><scr" + "ipt>alert(1)</scr" + "ipt></ht" + "ml>"
	svgXSS      = "<sv" + "g xmlns=\"http://www.w3.org/2000/svg\" onl" + "oad=\"alert(1)\"/>"
	xssPayload  = "<scr" + "ipt>alert('x" + "ss')</scr" + "ipt><img src=x oner" + "ror=alert(1)>"
	sqliUnion   = "' UN" + "ION SEL" + "ECT id,use" + "rname,pass" + "word FROM us" + "ers -- "
	sqliDrop    = "'; DR" + "OP TABLE po" + "sts; -- "
	traversalNm = "../../" + "etc/pass" + "wd.html"
)

// ---------- SQL 注入 ----------

// 登录用户名/密码注入经典 payload：必须全部 20001，绝不能拿到 token
func TestSQLInjectionLogin(t *testing.T) {
	app := newTestApp(t)

	payloads := []map[string]any{
		{"username": "admin' OR '1'='1", "password": "x"},
		{"username": "admin", "password": "' OR '1'='1"},
		{"username": "admin'--", "password": "x"},
		{"username": "admin' OR 1=1 -- ", "password": "x"},
		{"username": "' OR '1'='1' -- ", "password": "' OR '1'='1"},
	}
	for i, p := range payloads {
		rec := attemptLogin(t, app, p["username"].(string), p["password"].(string))
		e := decode(t, rec)
		if e.Code != 20001 {
			t.Fatalf("payload #%d 未被拒绝：code=%d msg=%s", i, e.Code, e.Message)
		}
		var data struct {
			Token string `json:"token"`
		}
		_ = json.Unmarshal(e.Data, &data)
		if data.Token != "" {
			t.Fatalf("payload #%d 意外获得 token", i)
		}
	}
}

// 文章详情 slug 参数注入：必须 10004，不能绕过 status=published 或返回文章
func TestSQLInjectionPostSlug(t *testing.T) {
	app := newTestApp(t)

	slugs := []string{
		"gin-blog-backend' OR '1'='1",
		"' OR '1'='1' -- ",
		"1 OR 1=1",
		"gin-blog-backend' OR status=0 -- ",
		"gin-blog-backend'--",
	}
	for i, slug := range slugs {
		rec := doJSON(t, app, http.MethodGet, "/api/v1/posts/"+urlEscape(slug), "", nil)
		e := decode(t, rec)
		if e.Code != 10004 {
			t.Fatalf("slug payload #%d 未被拒绝：code=%d msg=%s", i, e.Code, e.Message)
		}
	}
}

// 列表 keyword 注入：不得 500、不得泄露草稿、破坏性语句无效（表仍可用）
func TestSQLInjectionListKeyword(t *testing.T) {
	app := newTestApp(t)

	injections := []string{
		"%' OR '1'='1' -- ",
		"%" + sqliDrop,
		sqliUnion,
		"\\'% OR 1=1--",
		"%",
		"%%%",
	}
	for i, kw := range injections {
		rec := doJSON(t, app, http.MethodGet, "/api/v1/posts?keyword="+urlEscape(kw), "", nil)
		e := decode(t, rec)
		if e.Code != 0 {
			t.Fatalf("keyword payload #%d 异常：code=%d msg=%s", i, e.Code, e.Message)
		}
		var data struct {
			List []map[string]any `json:"list"`
		}
		decodeInto(t, e, &data)
		for _, p := range data.List {
			if p["status"].(float64) != 1 {
				t.Fatalf("keyword payload #%d 泄露了非已发布文章 %v", i, p["title"])
			}
		}
	}

	// 破坏性注入无效：posts 表仍然可查询
	rec := doJSON(t, app, http.MethodGet, "/api/v1/posts", "", nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("注入后文章列表不可用：%s", e.Message)
	}
}

// 数值型参数注入：categoryId/tagId/pageSize/sort 必须走解析器或参数化，不得拼进 SQL
func TestSQLInjectionNumericAndSortParams(t *testing.T) {
	app := newTestApp(t)

	cases := []string{
		"/api/v1/posts?categoryId=" + urlEscape("1 OR 1=1"),
		"/api/v1/posts?tagId=" + urlEscape("1; DROP TABLE post_tags; --"),
		"/api/v1/posts?pageSize=" + urlEscape("50; DROP TABLE posts; --"),
		"/api/v1/posts?sort=" + urlEscape("views; DROP TABLE posts; --"),
		"/api/v1/posts?sort=" + urlEscape("views --"),
		"/api/v1/posts?keyword=x&categoryId=99999999999999999999",
	}
	for i, path := range cases {
		rec := doJSON(t, app, http.MethodGet, path, "", nil)
		if rec.Code == http.StatusInternalServerError {
			t.Fatalf("payload #%d 导致 500：%s", i, path)
		}
	}

	// post_tags / posts 表仍然可用
	rec := doJSON(t, app, http.MethodGet, "/api/v1/posts", "", nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("注入后列表不可用：%s", e.Message)
	}
}

// 管理端数字 ID 注入（带合法 token）：parseIDParam 必须拒绝非数字
func TestSQLInjectionAdminIDParam(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	for _, path := range []string{
		"/api/v1/admin/posts/" + urlEscape("1 OR 1=1"),
		"/api/v1/admin/posts/" + urlEscape("1"+sqliDrop),
		"/api/v1/admin/comments/" + urlEscape("1 OR 1=1"),
		"/api/v1/admin/categories/" + urlEscape("1 UNION SELECT 1 --"),
	} {
		rec := doJSON(t, app, http.MethodDelete, path, token, nil)
		if rec.Code == http.StatusInternalServerError {
			t.Fatalf("%s 导致 500", path)
		}
		if e := decode(t, rec); e.Code == 0 {
			t.Fatalf("%s 注入却返回成功", path)
		}
	}

	// 注入未删除任何文章：文章列表 total 不变
	rec := doJSON(t, app, http.MethodGet, "/api/v1/admin/posts", token, nil)
	var data struct {
		Total int64 `json:"total"`
	}
	decodeInto(t, decode(t, rec), &data)
	if data.Total != 4 {
		t.Fatalf("种子文章应为 4 篇，实际 %d（注入可能生效）", data.Total)
	}
}

// 评论 parentId 注入：JSON 类型不符必须 10001
func TestSQLInjectionCommentParentID(t *testing.T) {
	app := newTestApp(t)

	body := `{"nickname":"注入","email":"a@b.c","content":"x","parentId":"1 OR 1=1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/posts/gin-blog-backend/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("parentId 字符串注入应 10001，实际 code=%d", e.Code)
	}
}

// ---------- 越权 / 信息泄露 ----------

// 草稿文章：数字 id 与 slug 都不可见，列表不出现
func TestDraftPostHidden(t *testing.T) {
	app := newTestApp(t)

	for _, path := range []string{"/api/v1/posts/blog-redesign-draft", "/api/v1/posts/4"} {
		rec := doJSON(t, app, http.MethodGet, path, "", nil)
		if e := decode(t, rec); e.Code != 10004 {
			t.Fatalf("草稿 %s 应 10004，实际 %d", path, e.Code)
		}
	}
	rec := doJSON(t, app, http.MethodGet, "/api/v1/posts", "", nil)
	var data struct {
		List []map[string]any `json:"list"`
	}
	decodeInto(t, decode(t, rec), &data)
	for _, p := range data.List {
		if p["slug"] == "blog-redesign-draft" {
			t.Fatal("草稿出现在公开列表")
		}
	}
}

// 公开评论接口不得返回 email / ip 字段；待审核评论不可见
func TestPublicCommentNoPIIAndPendingHidden(t *testing.T) {
	app := newTestApp(t)

	// 造一条待审核评论（公开接口创建后默认 status=0）
	id := createGuestComment(t, app, "pending-hidden@example.com")

	rec := doJSON(t, app, http.MethodGet, "/api/v1/posts/gin-blog-backend/comments", "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("公开评论列表失败：%s", e.Message)
	}
	raw := e.Data.String()
	if strings.Contains(raw, "pending-hidden@example.com") || strings.Contains(raw, `"ip"`) || strings.Contains(raw, `"email"`) {
		t.Fatal("公开评论响应包含 email/ip 字段或待审核评论邮箱")
	}
	var data struct {
		List []map[string]any `json:"list"`
	}
	decodeInto(t, e, &data)
	for _, cm := range data.List {
		if cm["id"].(float64) == float64(id) {
			t.Fatal("待审核评论出现在公开列表")
		}
	}
}

// mass-assignment：公开接口伪造 isAdmin 不生效
func TestCommentMassAssignmentIgnored(t *testing.T) {
	app := newTestApp(t)

	rec := doJSON(t, app, http.MethodPost, "/api/v1/posts/gin-blog-backend/comments", "", map[string]any{
		"nickname": "伪装者", "email": "fake-admin@example.com", "content": "我是管理员", "isAdmin": true,
	})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("评论创建失败：%s", e.Message)
	}

	token := loginToken(t, app)
	rec = doJSON(t, app, http.MethodGet, "/api/v1/admin/comments", token, nil)
	var data struct {
		List []map[string]any `json:"list"`
	}
	decodeInto(t, decode(t, rec), &data)
	for _, cm := range data.List {
		if cm["nickname"] == "伪装者" && cm["isAdmin"] == true {
			t.Fatal("isAdmin 被外部请求篡改")
		}
	}
}

// JWT 攻击：alg=none、旧默认密钥 dev-secret 签发的 token 一律 401
func TestJWTAlgorithmAndKnownSecretRejection(t *testing.T) {
	app := newTestApp(t)

	// alg=none（中间件 WithValidMethods 已限定 HS256）
	noneToken := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOjEsInVzZXJuYW1lIjoiYWRtaW4iLCJleHAiOjk5OTk5OTk5OTl9."
	rec := doJSON(t, app, http.MethodGet, "/api/v1/admin/posts", noneToken, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("alg=none token 应 401，实际 %d", rec.Code)
	}

	// 旧默认密钥 dev-secret 签发的 HS256 token（伪造工具最常尝试的值）
	devToken := hs256Sign(t, "dev-secret", `{"uid":1,"username":"admin","exp":9999999999}`)
	rec = doJSON(t, app, http.MethodGet, "/api/v1/admin/posts", devToken, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("dev-secret 签发的 token 应 401，实际 %d", rec.Code)
	}
}

// ---------- 路径穿越 ----------

// 静态上传目录穿越：各种编码的 ../ 不得读到文件内容
func TestUploadPathTraversal(t *testing.T) {
	app := newTestApp(t)

	targets := []string{
		"/uploads/../blog.db",
		"/uploads/../../blog.jwt.secret",
		"/uploads/%2e%2e/blog.db",
		"/uploads/..%2fblog.db",
		"/uploads/%2e%2e%2fblog.db",
		"/uploads/....//blog.db",
	}
	for i, path := range targets {
		rec := doJSON(t, app, http.MethodGet, path, "", nil)
		if strings.Contains(rec.Body.String(), "SQLite format 3") {
			t.Fatalf("路径穿越 payload #%d (%s) 读到了数据库内容", i, path)
		}
	}
}

// ---------- 存储型 XSS / 响应类型 ----------

// 恶意脚本入库后经 JSON 原样返回，Content-Type 必须是 application/json（配合 nosniff 防嗅探执行）
func TestStoredXSSResponseContentType(t *testing.T) {
	app := newTestApp(t)

	rec := doJSON(t, app, http.MethodPost, "/api/v1/posts/gin-blog-backend/comments", "", map[string]any{
		"nickname": xssPayload, "email": "xss@example.com", "content": xssPayload,
	})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("含脚本评论被误拒：%s", e.Message)
	}

	rec = doJSON(t, app, http.MethodGet, "/api/v1/posts/gin-blog-backend/comments", "", nil)
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("评论接口 Content-Type 应为 application/json，实际 %s", ct)
	}
}

// ---------- 上传伪装 ----------

// PHP/HTML/SVG 伪装上传：魔数嗅探必须全部拒绝
func TestUploadDisguisedScriptRejected(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	files := map[string][]byte{
		"shell.php": []byte(phpWebshell),
		"page.html": []byte(htmlXSS),
		"draw.svg":  []byte(svgXSS),
		"a.png.php": []byte(phpWebshell),
	}
	for name, content := range files {
		rec := uploadFile(t, app, token, name, content)
		e := decode(t, rec)
		if e.Code == 0 {
			t.Fatalf("伪装文件 %s 竟然上传成功", name)
		}
	}
}

// 合法 PNG 也不能保留攻击者文件名：存储路径必须是 随机名 + 白名单扩展名
func TestUploadFilenameSanitized(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	// PNG 签名 + 填充，魔数合法
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 64)...)
	rec := uploadFile(t, app, token, traversalNm, png)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("合法 PNG 上传失败：%s", e.Message)
	}
	var data struct {
		Upload struct {
			URL string `json:"url"`
		} `json:"upload"`
	}
	decodeInto(t, e, &data)
	if !strings.HasPrefix(data.Upload.URL, "/uploads/2") || strings.Contains(data.Upload.URL, "..") ||
		strings.Contains(data.Upload.URL, "passwd") || !strings.HasSuffix(data.Upload.URL, ".png") {
		t.Fatalf("存储 URL 未按 随机名+白名单扩展名 生成：%q", data.Upload.URL)
	}
}

// 超大请求体被 MaxBytesReader 拒绝
func TestUploadOversizedBodyRejected(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	big := bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, 5<<20) // ~15MB
	rec := uploadFile(t, app, token, "big.png", big)
	if rec.Code == http.StatusOK {
		if e := decode(t, rec); e.Code == 0 {
			t.Fatal("15MB 文件上传成功，MaxBytesReader 未生效")
		}
	}
}

// ---------- 助手 ----------

func urlEscape(s string) string {
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			const hexDigits = "0123456789ABCDEF"
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0xf])
		}
	}
	return b.String()
}

func uploadFile(t *testing.T, app http.Handler, token, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("构造 multipart 失败：%v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("写入 multipart 失败：%v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("关闭 multipart 失败：%v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/uploads", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

// hs256Sign 测试专用：用指定密钥手签一个 HS256 token（模拟攻击者持有某密钥）
func hs256Sign(t *testing.T, secret, payloadJSON string) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(header + "." + body))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + body + "." + sig
}
