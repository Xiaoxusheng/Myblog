package main

// 操作日志（#76）+ 备份（#77-80）+ 导出/导入（#81/#82）测试

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"myblog/server/config"
	"myblog/server/model"
)

func TestAuditLogs(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	post := createPost(t, r, token, map[string]any{"title": "审计文"})
	updatePostHelper(t, r, token, post.ID, map[string]any{"title": "审计文改"})

	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs", token, nil)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("审计列表失败：%s", e.Message)
	}
	var data struct {
		List []struct {
			Action string `json:"action"`
			IPHash string `json:"ipHash"`
		} `json:"list"`
		Total int64 `json:"total"`
	}
	decodeInto(t, e, &data)
	if data.Total < 2 {
		t.Fatalf("应至少有 create+update 两条日志，got %d", data.Total)
	}
	if data.List[0].Action != "post.update" || data.List[1].Action != "post.create" {
		t.Fatalf("最新应为 post.update / post.create，got %s / %s", data.List[0].Action, data.List[1].Action)
	}
	if data.List[0].IPHash == "" {
		t.Fatal("ipHash 应非空（哈希化存储）")
	}
	// 前缀过滤
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs?action=post.update", token, nil)
	decodeInto(t, decode(t, rec), &data)
	if data.Total != 1 {
		t.Fatalf("前缀过滤应只命中 1 条，got %d", data.Total)
	}
}

func TestBackupLifecycle(t *testing.T) {
	t.Setenv("BLOG_BACKUP_DIR", t.TempDir())
	dbPath := filepath.Join(t.TempDir(), "blog.db")
	r := newTestAppCfg(t, func(cfg *config.Config) {
		cfg.DBPath = dbPath
	})
	token := loginToken(t, r)

	// 创建数据库备份
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/backups", token, map[string]any{"type": "database"})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建备份失败：%s", e.Message)
	}
	var created struct {
		Item struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"item"`
	}
	decodeInto(t, e, &created)
	if !strings.HasSuffix(created.Item.Name, ".db") || created.Item.Size == 0 {
		t.Fatalf("备份文件异常：%+v", created.Item)
	}

	// 非法文件名防穿越（单段路径内含 ..，命中显式拒绝分支）
	if e := decode(t, doJSON(t, r, http.MethodDelete, "/api/v1/admin/backups/a..b", token, nil)); e.Code == 0 {
		t.Fatal("路径穿越应被拒绝")
	}

	// 列表
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/backups", token, nil)
	var list struct {
		List []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"list"`
	}
	decodeInto(t, decode(t, rec), &list)
	if len(list.List) != 1 || list.List[0].Type != "database" {
		t.Fatalf("备份列表不符：%+v", list.List)
	}

	// 下载内容为合法 SQLite 头
	rec = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/backups/%s/download", list.List[0].Name), token, nil)
	if rec.Code != http.StatusOK || rec.Body.Len() < 16 {
		t.Fatalf("下载异常：code=%d len=%d", rec.Code, rec.Body.Len())
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("SQLite format 3")) {
		t.Fatal("备份应为 SQLite 文件")
	}

	// 删除
	rec = doJSON(t, r, http.MethodDelete, fmt.Sprintf("/api/v1/admin/backups/%s", list.List[0].Name), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除备份失败：%s", e.Message)
	}
	if e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/backups", token, nil)); e.Code != 0 {
		t.Fatal("列表失败")
	}
}

func TestExportImportRoundtrip(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	p1 := createPost(t, r, token, map[string]any{"title": "导出文", "slug": "export-post", "status": 1})
	createPost(t, r, token, map[string]any{"title": "导出文二", "slug": "export-post-2", "status": 1})
	base := baseTotal(t, r)

	// 导出
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/export", token, nil)
	if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Fatalf("导出失败：code=%d", rec.Code)
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("导出不是合法 zip：%v", err)
	}
	var exportPosts []map[string]any
	for _, zf := range zr.File {
		if zf.Name == "posts.json" {
			rc, _ := zf.Open()
			if err := json.NewDecoder(rc).Decode(&exportPosts); err != nil {
				t.Fatalf("posts.json 解析失败：%v", err)
			}
			rc.Close()
		}
	}
	if len(exportPosts) < 2 {
		t.Fatalf("导出文章数应 ≥2，got %d", len(exportPosts))
	}
	found := false
	for _, p := range exportPosts {
		if p["slug"] == "export-post" && p["title"] == "导出文" {
			found = true
		}
	}
	if !found {
		t.Fatal("导出内容缺 export-post")
	}

	// dryRun 预览：全部 slug 冲突
	previewRec := multipartImport(t, r, token, rec.Body.Bytes(), "true", "skip")
	previewEnv := decode(t, previewRec)
	if previewEnv.Code != 0 {
		t.Fatalf("预览失败：%s", previewEnv.Message)
	}
	var preview struct {
		Conflicts []ginH `json:"conflicts"`
	}
	decodeInto(t, previewEnv, &preview)
	if len(preview.Conflicts) == 0 {
		t.Fatal("预览应发现 slug 冲突")
	}

	// skip 导入：数量不变
	if e := decode(t, multipartImport(t, r, token, rec.Body.Bytes(), "false", "skip")); e.Code != 0 {
		t.Fatalf("skip 导入失败：%s", e.Message)
	}
	if got := baseTotal(t, r); got != base {
		t.Fatalf("skip 导入后公开文章数不应变，base=%d got=%d", base, got)
	}

	// update 导入：数量仍不变但内容更新
	for _, p := range exportPosts {
		if p["slug"] == "export-post" {
			p["title"] = "导出文-更新"
		}
	}
	repacked := repackPosts(t, exportPosts)
	if e := decode(t, multipartImport(t, r, token, repacked, "false", "update")); e.Code != 0 {
		t.Fatalf("update 导入失败：%s", e.Message)
	}
	rec = doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/posts/%d", p1.ID), token, nil)
	var detail struct {
		Post model.AdminPostItemDTO `json:"post"`
	}
	decodeInto(t, decode(t, rec), &detail)
	if detail.Post.Title != "导出文-更新" {
		t.Fatalf("update 导入后标题应更新，got %s", detail.Post.Title)
	}
}

// multipartImport 构造 multipart 上传导入
func multipartImport(t *testing.T, r http.Handler, token string, zipBytes []byte, dryRun, strategy string) *httptest.ResponseRecorder {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile("file", "export.zip")
	if err != nil {
		t.Fatalf("构造表单失败：%v", err)
	}
	if _, err := fw.Write(zipBytes); err != nil {
		t.Fatalf("写入表单失败：%v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("关闭表单失败：%v", err)
	}

	path := "/api/v1/admin/import?strategy=" + strategy
	if dryRun == "true" {
		path += "&dryRun=true"
	}
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

type ginH = map[string]any

// repackPosts 将修改后的 posts 数组重新打成仅含 posts.json 的导入包
func repackPosts(t *testing.T, posts []map[string]any) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	w, err := zw.Create("posts.json")
	if err != nil {
		t.Fatalf("创建 zip 条目失败：%v", err)
	}
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		t.Fatalf("编码 posts 失败：%v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("关闭 zip 失败：%v", err)
	}
	return buf.Bytes()
}
