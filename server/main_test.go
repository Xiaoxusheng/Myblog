package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"myblog/server/config"
	"myblog/server/middleware"
	"myblog/server/model"
	"myblog/server/router"

	"github.com/gin-gonic/gin"
)

const testJWTSecret = "test-secret-key"

// newTestApp 每个用例独立的 SQLite 内存库 + 完整路由 + 幂等种子数据
func newTestApp(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	middleware.ResetCommentRateLimit()

	cfg := &config.Config{
		Port:      "0",
		JWTSecret: testJWTSecret,
		DBType:    "sqlite",
		DBPath:    fmt.Sprintf("file:memdb%d?mode=memory&cache=shared", time.Now().UnixNano()),
		UploadDir: t.TempDir(),
	}
	db, err := model.Open(cfg)
	if err != nil {
		t.Fatalf("打开内存库失败：%v", err)
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

// ---------- 请求/解码助手 ----------

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

func loginToken(t *testing.T, r http.Handler) string {
	t.Helper()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/auth/login", "",
		map[string]any{"username": "admin", "password": "admin123"})
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
