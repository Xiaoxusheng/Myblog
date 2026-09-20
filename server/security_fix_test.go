package main

// 模块六安全修复回归测试：
// 1. 备份/导出递归包含 uploads 子目录（YYYYMM/）——修复媒体丢失
// 2. SQLite 数据库备份为 VACUUM INTO 一致性快照（可独立打开查询）
// 3. 导入媒体按包内相对路径落盘；穿越条目（media/../x、反斜杠变体、盘符）被拒绝
// 4. 改密后旧 JWT 立即吊销（token_version），改密响应返回新 token
// 5. 操作日志 action 前缀过滤转义 LIKE 通配符

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"myblog/server/config"
	"myblog/server/model"

	_ "github.com/glebarez/go-sqlite"
)

// writeTestFile 写测试文件（自动建父目录）
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("创建目录失败：%v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写文件失败：%v", err)
	}
}

// zipEntryNames 读取磁盘上 zip 文件的全部条目名
func zipEntryNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 zip 失败：%v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("不是合法 zip：%v", err)
	}
	names := make(map[string]bool, len(zr.File))
	for _, zf := range zr.File {
		names[zf.Name] = true
	}
	return names
}

// buildZip 内存构造导入 zip 包
func buildZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("创建 zip 条目失败：%v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("写入 zip 条目失败：%v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("关闭 zip 失败：%v", err)
	}
	return buf.Bytes()
}

// TestBackupAndExportIncludeMediaSubdirs 全量备份与全站导出应递归包含 uploads 子目录
func TestBackupAndExportIncludeMediaSubdirs(t *testing.T) {
	t.Setenv("BLOG_BACKUP_DIR", t.TempDir())
	uploadDir := t.TempDir()
	r := newTestAppCfg(t, func(cfg *config.Config) { cfg.UploadDir = uploadDir })
	token := loginToken(t, r)

	// 模拟媒体库真实布局：根文件 + YYYYMM/ 子目录
	writeTestFile(t, filepath.Join(uploadDir, "root.txt"), "root")
	writeTestFile(t, filepath.Join(uploadDir, "202609", "nested.png"), "img")

	// 全量备份
	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/backups", token, map[string]any{"type": "full"})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建全量备份失败：%s", e.Message)
	}
	var created struct {
		Item struct {
			Name string `json:"name"`
		} `json:"item"`
	}
	decodeInto(t, e, &created)

	names := zipEntryNames(t, filepath.Join(os.Getenv("BLOG_BACKUP_DIR"), created.Item.Name))
	if !names["media/root.txt"] {
		t.Fatalf("全量备份缺 media/root.txt，got %v", names)
	}
	if !names["media/202609/nested.png"] {
		t.Fatalf("全量备份应含子目录条目 media/202609/nested.png，got %v", names)
	}
	if !names["database/blog.db"] {
		t.Fatalf("全量备份缺数据库条目，got %v", names)
	}

	// 全站导出（复用同一递归打包函数，验证导出路径）
	rec = doJSON(t, r, http.MethodGet, "/api/v1/admin/export", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("导出失败：code=%d", rec.Code)
	}
	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("导出不是合法 zip：%v", err)
	}
	exportNames := map[string]bool{}
	for _, zf := range zr.File {
		exportNames[zf.Name] = true
	}
	if !exportNames["media/202609/nested.png"] {
		t.Fatalf("导出应含子目录条目 media/202609/nested.png，got %v", exportNames)
	}
}

// TestDatabaseBackupVACUUMSnapshot 数据库备份必须是可独立打开查询的一致性快照
func TestDatabaseBackupVACUUMSnapshot(t *testing.T) {
	if os.Getenv("BLOG_TEST_MYSQL_DSN") != "" {
		t.Skip("MySQL 模式无数据库文件备份（契约明确报错）")
	}
	t.Setenv("BLOG_BACKUP_DIR", t.TempDir())
	r := newTestApp(t)
	token := loginToken(t, r)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/admin/backups", token, map[string]any{"type": "database"})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("创建备份失败：%s", e.Message)
	}
	var created struct {
		Item struct {
			Name string `json:"name"`
		} `json:"item"`
	}
	decodeInto(t, e, &created)

	// VACUUM INTO 产物应能独立打开并查询（直接 copy 热库可能页级不一致）
	db, err := sql.Open("sqlite", filepath.Join(os.Getenv("BLOG_BACKUP_DIR"), created.Item.Name))
	if err != nil {
		t.Fatalf("快照无法打开：%v", err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&n); err != nil {
		t.Fatalf("快照查询失败：%v", err)
	}
	if n < 1 {
		t.Fatal("快照应包含种子管理员")
	}
}

// TestImportMediaNestedAndTraversal 导入媒体应按包内相对路径落盘并拒绝穿越条目
func TestImportMediaNestedAndTraversal(t *testing.T) {
	uploadDir := t.TempDir()
	r := newTestAppCfg(t, func(cfg *config.Config) { cfg.UploadDir = uploadDir })
	token := loginToken(t, r)

	zipBytes := buildZip(t, map[string]string{
		"posts.json":           `[]`,
		"media/root2.txt":      "root2",
		"media/202609/n.png":   "png",
		"media/../evil.txt":    "evil",
		"media/..\\evil2.txt":  "evil2",
		"media/202609/../../e": "e",
		"media/C:/win.txt":     "win",
	})
	rec := multipartImport(t, r, token, zipBytes, "false", "skip")
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("导入失败：%s", e.Message)
	}

	// 合法条目落盘（含子目录）
	assertFileContent(t, filepath.Join(uploadDir, "root2.txt"), "root2")
	assertFileContent(t, filepath.Join(uploadDir, "202609", "n.png"), "png")

	// 穿越条目：uploads 内外均不得出现
	for _, p := range []string{
		filepath.Join(uploadDir, "evil.txt"),
		filepath.Join(uploadDir, "evil2.txt"),
		filepath.Join(uploadDir, "e"),
		filepath.Join(uploadDir, "..", "evil.txt"),
		filepath.Join(uploadDir, "..", "evil2.txt"),
	} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("穿越条目不得落盘：%s", p)
		}
	}

	// summary.media 只计合法条目（2 个）
	var data struct {
		Summary map[string]int `json:"summary"`
	}
	decodeInto(t, e, &data)
	if data.Summary["media"] != 2 {
		t.Fatalf("合法媒体应计 2 个，got %d", data.Summary["media"])
	}
}

// assertFileContent 断言文件存在且内容一致
func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("文件应存在：%s：%v", path, err)
	}
	if string(got) != want {
		t.Fatalf("文件内容不符：%s：got %q want %q", path, got, want)
	}
}

// TestPasswordChangeRevokesOldToken 改密后旧 JWT 立即失效，响应携带新 token
func TestPasswordChangeRevokesOldToken(t *testing.T) {
	r := newTestApp(t)
	oldToken := loginToken(t, r)

	// 改密前旧 token 有效
	if e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/auth/me", oldToken, nil)); e.Code != 0 {
		t.Fatalf("改密前 me 应成功：%s", e.Message)
	}

	rec := doJSON(t, r, http.MethodPut, "/api/v1/admin/auth/password", oldToken,
		map[string]any{"oldPassword": "admin123", "newPassword": "newpass456"})
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("改密失败：%s", e.Message)
	}
	var data struct {
		Token string `json:"token"`
	}
	decodeInto(t, e, &data)
	if data.Token == "" {
		t.Fatal("改密成功应返回新 token")
	}

	// 旧 token 立即失效（10002 = 未授权）
	if e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/auth/me", oldToken, nil)); e.Code != 10002 {
		t.Fatalf("改密后旧 token 应失效（10002），got code=%d", e.Code)
	}
	// 响应中的新 token 有效
	if e := decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/auth/me", data.Token, nil)); e.Code != 0 {
		t.Fatalf("新 token 应有效：%s", e.Message)
	}
}

// TestAuditLogsActionPrefixEscape action 前缀过滤应转义 LIKE 通配符
func TestAuditLogsActionPrefixEscape(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)

	// 造两条特殊 action：prd（正常前缀）与 pr%（字面含通配符）
	if err := model.CreateAuditLog(model.DB, "prd", "t", "1", "", ""); err != nil {
		t.Fatalf("写日志失败：%v", err)
	}
	if err := model.CreateAuditLog(model.DB, "pr%", "t", "2", "", ""); err != nil {
		t.Fatalf("写日志失败：%v", err)
	}

	var data struct {
		Total int `json:"total"`
		List  []struct {
			Action string `json:"action"`
		} `json:"list"`
	}

	// action=prd：正常前缀语义，恰好 1 条
	decodeInto(t, decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs?action=prd", token, nil)), &data)
	if data.Total != 1 || len(data.List) != 1 || data.List[0].Action != "prd" {
		t.Fatalf("action=prd 应命中 1 条，got total=%d list=%+v", data.Total, data.List)
	}

	// action=pr%（URL 编码 pr%25）：转义后只命中字面 "pr%" 1 条；
	// 若未转义（LIKE 'pr%%'）则 "prd" 也会被命中
	decodeInto(t, decode(t, doJSON(t, r, http.MethodGet, "/api/v1/admin/audit-logs?action=pr%25", token, nil)), &data)
	if data.Total != 1 || len(data.List) != 1 || data.List[0].Action != "pr%" {
		t.Fatalf("action=pr%% 应转义后仅命中字面 1 条，got total=%d list=%+v", data.Total, data.List)
	}

	// 分页参数与转义组合不报错
	if e := decode(t, doJSON(t, r, http.MethodGet, fmt.Sprintf("/api/v1/admin/audit-logs?action=%s&page=1&pageSize=5", "pr_"), token, nil)); e.Code != 0 {
		t.Fatalf("含下划线前缀查询应正常：%s", e.Message)
	}
}
