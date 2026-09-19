package main

// llms.txt（AEO）与系统健康（#83/#84）测试

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"myblog/server/config"
	"myblog/server/handler"
)

func TestLlmsTxt(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	createPost(t, r, token, map[string]any{
		"title": "AI 可见文章", "slug": "llms-post", "status": 1, "summary": "给 AI 看的摘要",
	})
	// 定时发布文章不应出现在 llms.txt
	createPost(t, r, token, map[string]any{
		"title": "未发布", "status": 3, "publishAt": time.Now().Add(24 * time.Hour),
	})

	body := doJSON(t, r, http.MethodGet, "/llms.txt", "", nil).Body.String()

	if !strings.Contains(body, "# My Blog") && !strings.Contains(body, "# ") {
		t.Fatalf("llms.txt 应含站点标题，got:\n%s", body[:min(len(body), 200)])
	}
	if !strings.Contains(body, "AI 可见文章") || !strings.Contains(body, "给 AI 看的摘要") {
		t.Fatal("llms.txt 应含已发布文章与摘要")
	}
	if strings.Contains(body, "未发布") {
		t.Fatal("llms.txt 不应包含非已发布文章")
	}
	if !strings.Contains(body, "/post/llms-post") {
		t.Fatal("llms.txt 应含文章 URL")
	}
}

func TestHealth(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	handler.SetVersion("v1.2.0-test")
	e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/health", token, nil))
	if e.Code != 0 {
		t.Fatalf("health 失败：%s", e.Message)
	}
	var data struct {
		Version      string `json:"version"`
		GoVersion    string `json:"goVersion"`
		DBType       string `json:"dbType"`
		DBStatus     string `json:"dbStatus"`
		PostCount    int64  `json:"postCount"`
		CommentCount int64  `json:"commentCount"`
		MediaCount   int64  `json:"mediaCount"`
		UploadSize   int64  `json:"uploadSize"`
	}
	decodeInto(t, e, &data)
	if data.Version != "v1.2.0-test" {
		t.Fatalf("version 应注入，got %s", data.Version)
	}
	if !strings.HasPrefix(data.GoVersion, "go") {
		t.Fatalf("goVersion 异常：%s", data.GoVersion)
	}
	if data.DBType != "sqlite" || data.DBStatus != "ok" {
		t.Fatalf("DB 信息异常：%s/%s", data.DBType, data.DBStatus)
	}
	if data.PostCount == 0 {
		t.Fatal("postCount 应包含种子数据")
	}
}

func TestAuditLogFromBackupNotify(t *testing.T) {
	// 通知中心在备份后会产生 backup 类型通知（Phase5 备份链路复验）
	t.Setenv("BLOG_BACKUP_DIR", t.TempDir())
	r := newTestAppCfg(t, func(cfg *config.Config) {
		cfg.DBPath = filepath.Join(t.TempDir(), "blog.db")
	})
	token := loginToken(t, r)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/backups", token, map[string]any{"type": "database"})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("备份失败：%s", e.Message)
	}

	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/notifications", token, nil)
	var data struct {
		List []struct {
			Type string `json:"type"`
		} `json:"list"`
	}
	decodeInto(t, decode(t, rec), &data)
	found := false
	for _, n := range data.List {
		if n.Type == "backup" {
			found = true
		}
	}
	if !found {
		t.Fatal("备份后应有 backup 通知")
	}
}
