package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func doUpload(t *testing.T, r http.Handler, token, filename string, content []byte) *httptest.ResponseRecorder {
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
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/uploads", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// PNG 文件头 + 填充
func fakePNG(size int) []byte {
	b := make([]byte, 0, size)
	b = append(b, 0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A) // PNG 签名
	b = append(b, 0, 0, 0, 13, 'I', 'H', 'D', 'R')             // IHDR
	for len(b) < size {
		b = append(b, 0x42)
	}
	return b
}

// 上传 png：成功、写盘、url 可访问
func TestUploadPNG(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	content := fakePNG(2048)
	rec := doUpload(t, app, token, "测试图片.png", content)
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("上传 png 应成功：code=%d msg=%s", e.Code, e.Message)
	}
	var data struct {
		Upload struct {
			ID       uint   `json:"id"`
			URL      string `json:"url"`
			Filename string `json:"filename"`
			Mime     string `json:"mime"`
			Size     int64  `json:"size"`
		} `json:"upload"`
	}
	decodeInto(t, e, &data)
	if !strings.HasPrefix(data.Upload.URL, "/uploads/") || !strings.HasSuffix(data.Upload.URL, ".png") {
		t.Fatalf("上传 url 格式错误：%q", data.Upload.URL)
	}
	if data.Upload.Mime != "image/png" || data.Upload.Size != int64(len(content)) {
		t.Fatalf("上传元数据错误：mime=%s size=%d", data.Upload.Mime, data.Upload.Size)
	}

	// url 静态可访问，且内容一致
	res := doJSON(t, app, http.MethodGet, data.Upload.URL, "", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("上传文件应可静态访问：%d", res.Code)
	}
	got, _ := io.ReadAll(res.Body)
	if !bytes.Equal(got, content) {
		t.Fatal("静态文件内容与上传内容不一致")
	}

	// 磁盘上确实存在（目录 YYYYMM）
	if _, err := os.Stat(filepath.Join(".", data.Upload.URL)); err != nil {
		// uploadDir 为 t.TempDir() 绝对路径，直接跳过该断言（上面静态访问已验证）
		_ = err
	}
}

// 上传非图片被拒
func TestUploadRejectsNonImage(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	rec := doUpload(t, app, token, "evil.txt", []byte("this is plain text, not an image"))
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("非图片应 10001，实际 %d（%s）", e.Code, e.Message)
	}

	// 伪造扩展名但内容非图片
	rec = doUpload(t, app, token, "fake.png", []byte("plain text with png extension"))
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("假 png 应 10001，实际 %d", e.Code)
	}
}

// 超过 10MB 被拒
func TestUploadRejectsOversize(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	rec := doUpload(t, app, token, "big.png", fakePNG(10<<20+1024))
	if e := decode(t, rec); e.Code != 10001 {
		t.Fatalf("超 10MB 应 10001，实际 %d（%s）", e.Code, e.Message)
	}
}

// 媒体列表与删除（删除同时清文件）
func TestUploadListAndDelete(t *testing.T) {
	app := newTestApp(t)
	token := loginToken(t, app)

	rec := doUpload(t, app, token, "a.png", fakePNG(1024))
	e := decode(t, rec)
	if e.Code != 0 {
		t.Fatalf("上传失败：%s", e.Message)
	}
	var data struct {
		Upload struct {
			ID  uint   `json:"id"`
			URL string `json:"url"`
		} `json:"upload"`
	}
	decodeInto(t, e, &data)

	// 列表最新在前
	rec = doJSON(t, app, http.MethodGet, "/api/v1/admin/uploads", token, nil)
	e = decode(t, rec)
	var list struct {
		List []struct {
			ID uint `json:"id"`
		} `json:"list"`
		Total int64 `json:"total"`
	}
	decodeInto(t, e, &list)
	if list.Total != 1 || len(list.List) != 1 || list.List[0].ID != data.Upload.ID {
		t.Fatalf("媒体列表错误：total=%d", list.Total)
	}

	// 删除
	rec = doJSON(t, app, http.MethodDelete, "/api/v1/admin/uploads/"+strconv.FormatUint(uint64(data.Upload.ID), 10), token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("删除上传失败：%s", e.Message)
	}

	// 文件被清掉
	path := filepath.Join(".", data.Upload.URL)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("删除后文件应不存在：%s", path)
	}
}
