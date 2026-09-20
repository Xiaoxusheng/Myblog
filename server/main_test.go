package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"myblog/server/config"
	"myblog/server/handler"
	"myblog/server/middleware"
	"myblog/server/model"
	"myblog/server/router"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

const testJWTSecret = "test-secret-key"

// testCryptoKeyHex 32 字节加密密钥（hex），供加密相关用例注入
const testCryptoKeyHex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// newTestApp 每个用例独立的 SQLite 内存库 + 完整路由 + 幂等种子数据
func newTestApp(t *testing.T) *gin.Engine {
	t.Helper()
	return newTestAppCfg(t, nil)
}

// newTestAppCfg 同 newTestApp，tune 允许用例在 Open 前调整配置（如注入加密密钥）
func newTestAppCfg(t *testing.T, tune func(*config.Config)) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	middleware.ResetCommentRateLimit()
	middleware.ResetLoginRateLimit()
	handler.ResetTrackCounters()

	cfg := &config.Config{
		Port:      "0",
		JWTSecret: testJWTSecret,
		DBType:    "sqlite",
		DBPath:    fmt.Sprintf("file:memdb%d?mode=memory&cache=shared", time.Now().UnixNano()),
		UploadDir: t.TempDir(),
	}
	// 可选：把整套测试跑在真实 MySQL 上（本地 SQLite 发现不了保留字之类的问题）。
	//
	// 用法：BLOG_TEST_MYSQL_DSN='root:pw@tcp(host:3306)/myblog_verify?charset=utf8mb4&parseTime=True&loc=Local' \
	//         go test ./...
	//
	// 原理：DSN 里的库名只作为「可连接的管理库」，每个用例会在同一 MySQL 实例上
	// 建一个独立库 myblog_verify_<随机>，用完即删。这样各用例互不干扰，可并行。
	// **绝不可**把 DSN 指向生产库 —— 虽然用例只操作自己新建的库，但误配代价太高。
	if dsn := strings.TrimSpace(os.Getenv("BLOG_TEST_MYSQL_DSN")); dsn != "" {
		cfg.DBType = "mysql"
		cfg.MySQLDSN = dsn
	}
	if tune != nil {
		tune(cfg)
	}
	if cfg.DBType == "mysql" {
		name, err := createMySQLTestDB(cfg.MySQLDSN)
		if err != nil {
			t.Fatalf("创建 MySQL 用例库失败：%v", err)
		}
		cfg.MySQLDSN = replaceDSNDatabase(cfg.MySQLDSN, name)
		t.Cleanup(func() { dropMySQLDatabase(cfg.MySQLDSN, name) })
	}
	db, err := model.Open(cfg)
	if err != nil {
		t.Fatalf("打开数据库失败：%v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("迁移失败：%v", err)
	}
	if err := model.Seed(db); err != nil {
		t.Fatalf("seed 失败：%v", err)
	}
	return router.Setup(cfg)
}

// ---------- MySQL 用例库管理（仅 BLOG_TEST_MYSQL_DSN 模式使用）----------

var mysqlTestDBSeq atomic.Int64

// createMySQLTestDB 在同一实例上建一个独立库并返回库名。
// 用「连到管理库 → CREATE DATABASE」的方式，避免依赖固定库名。
func createMySQLTestDB(adminDSN string) (string, error) {
	name := fmt.Sprintf("myblog_verify_%d_%d", time.Now().UnixNano()%1e9, mysqlTestDBSeq.Add(1))
	sqlDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		return "", err
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return "", err
	}
	return name, nil
}

// dropMySQLDatabase 删除用例库；失败只记日志，不影响用例结果。
func dropMySQLDatabase(adminDSN, name string) {
	sqlDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		return
	}
	defer sqlDB.Close()
	_, _ = sqlDB.Exec("DROP DATABASE IF EXISTS `" + name + "`")
}

// replaceDSNDatabase 把 DSN 中的库名替换为新库名。
// go-sql-driver 的 DSN 形如 user:pass@tcp(host:port)/dbname?params
func replaceDSNDatabase(dsn, dbName string) string {
	slash := strings.LastIndex(dsn, "/")
	if slash < 0 {
		return dsn
	}
	rest := dsn[slash+1:]
	if q := strings.Index(rest, "?"); q >= 0 {
		return dsn[:slash+1] + dbName + rest[q:]
	}
	return dsn[:slash+1] + dbName
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func doJSON(t *testing.T, r http.Handler, method, path, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("编码请求体失败：%v", err)
		}
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var e envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("响应不是合法 JSON：%v\nbody: %s", err, rec.Body.String())
	}
	return e
}

// decodeInto 将 data 解析到目标结构
func decodeInto(t *testing.T, e envelope, v any) {
	t.Helper()
	if err := json.Unmarshal(e.Data, v); err != nil {
		t.Fatalf("解析 data 失败：%v\ndata: %s", err, string(e.Data))
	}
}

// attemptLogin 走完整验证码流程的登录（#11/#86），返回原始响应
func attemptLogin(t *testing.T, r http.Handler, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	capRec := doJSON(t, r, http.MethodGet, "/api/v1/admin/auth/captcha", "", nil)
	capEnv := decode(t, capRec)
	var capData struct {
		CaptchaID string `json:"captchaId"`
	}
	decodeInto(t, capEnv, &capData)
	captchaCode := handler.GetCaptchaAnswer(capData.CaptchaID)
	return doJSON(t, r, http.MethodPost, "/api/v1/admin/auth/login", "",
		map[string]any{
			"username": username, "password": password,
			"captchaId": capData.CaptchaID, "captchaCode": captchaCode,
		})
}

func loginToken(t *testing.T, r http.Handler) string {
	t.Helper()
	rec := attemptLogin(t, r, "admin", "admin123")
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("登录失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Token string `json:"token"`
		User  struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	decodeInto(t, e, &data)
	if data.Token == "" {
		t.Fatal("token 为空")
	}
	if data.User.Username != "admin" {
		t.Fatalf("登录用户错误：%s", data.User.Username)
	}
	return data.Token
}

// createPost 通过管理端接口创建文章，返回契约 AdminPostItem
func createPost(t *testing.T, r http.Handler, token string, fields map[string]any) model.AdminPostItemDTO {
	t.Helper()
	payload := map[string]any{
		"title":   "默认标题",
		"slug":    "",
		"summary": "默认摘要",
		"content": "默认正文内容",
		"cover":   "",
		"status":  1,
		"isTop":   false,
		"tags":    []string{},
	}
	for k, v := range fields {
		payload[k] = v
	}
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/posts", token, payload)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建文章失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Post model.AdminPostItemDTO `json:"post"`
	}
	decodeInto(t, e, &data)
	return data.Post
}

type postListData struct {
	List     []model.PostSummaryDTO `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}

func publicPostList(t *testing.T, r http.Handler, query string) postListData {
	t.Helper()
	rec := doJSON(t, r, http.MethodGet, "/api/v1/posts"+query, "", nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("公开列表失败：code=%d msg=%s", e.Code, e.Message)
	}
	var data postListData
	decodeInto(t, e, &data)
	return data
}

func baseTotal(t *testing.T, r http.Handler) int64 {
	t.Helper()
	return publicPostList(t, r, "?page=1&pageSize=50").Total
}
