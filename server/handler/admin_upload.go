package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// maxUploadSize 上传上限：10MB（契约 #45）
const maxUploadSize = 10 << 20

// allowedImageExt 由嗅探出的 MIME 映射扩展名（白名单，杜绝任意文件写入；SVG 含脚本风险，不放开）
var allowedImageExt = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
	"image/bmp":  ".bmp",
}

// AdminUpload POST /api/v1/admin/uploads —— multipart 字段 file，仅图片 ≤10MB
func AdminUpload(c *gin.Context) {
	// 请求体上限 = 10MB 文件 + multipart 编码开销
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize+(64<<10))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.Fail(c, common.CodeParamError, "上传失败：请通过 multipart 字段 file 提交且不超过 10MB")
		return
	}
	if fileHeader.Size > maxUploadSize {
		common.Fail(c, common.CodeParamError, "图片大小不能超过 10MB")
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		common.Fail(c, common.CodeParamError, "无法读取上传文件")
		return
	}
	defer func() { _ = src.Close() }()

	// 魔数嗅探（扩展名不可信）
	head := make([]byte, 512)
	n, err := io.ReadAtLeast(src, head, 8)
	if err != nil && err != io.ErrUnexpectedEOF {
		common.Fail(c, common.CodeParamError, "无法读取上传文件")
		return
	}
	head = head[:n]
	mime := http.DetectContentType(head)
	ext, ok := allowedImageExt[mime]
	if !ok || !strings.HasPrefix(mime, "image/") {
		common.Fail(c, common.CodeParamError, "仅支持上传图片文件（png/jpg/gif/webp/bmp）")
		return
	}

	// 原始文件名入库展示用（截断到 255，兼容 varchar(255)）
	filename := fileHeader.Filename
	if !utf8.ValidString(filename) {
		filename = "upload" + ext
	}
	if utf8.RuneCountInString(filename) > 255 {
		filename = string([]rune(filename)[:255])
	}

	// 上传目录：uploads/YYYYMM/随机名.扩展名
	monthDir := time.Now().Format("200601")
	absDir := filepath.Join(uploadDir, monthDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		common.ServerError(c, err)
		return
	}
	name := randomHex(16) + ext
	absPath := filepath.Join(absDir, name)

	dst, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		common.ServerError(c, err)
		return
	}
	// 先写回嗅探时消费的字节，再流式拷贝剩余部分（不整文件读入内存）
	if _, err := dst.Write(head); err != nil {
		_ = dst.Close()
		_ = os.Remove(absPath)
		common.ServerError(c, err)
		return
	}
	size, err := io.Copy(dst, src)
	if err != nil {
		_ = dst.Close()
		_ = os.Remove(absPath)
		common.ServerError(c, err)
		return
	}
	size += int64(len(head))
	if err := dst.Close(); err != nil {
		_ = os.Remove(absPath)
		common.ServerError(c, err)
		return
	}
	if size > maxUploadSize { // MaxBytesReader 之外的二次保险
		_ = os.Remove(absPath)
		common.Fail(c, common.CodeParamError, "图片大小不能超过 10MB")
		return
	}

	upload := model.Upload{
		Filename: filename,
		Path:     monthDir + "/" + name,
		URL:      "/uploads/" + monthDir + "/" + name,
		Size:     size,
		Mime:     mime,
	}
	if err := model.DB.Create(&upload).Error; err != nil {
		_ = os.Remove(absPath)
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"upload": upload})
}

// AdminListUploads GET /api/v1/admin/uploads —— 最新在前
func AdminListUploads(c *gin.Context) {
	pq := common.ParsePage(c, 10)

	var total int64
	if err := model.DB.Model(&model.Upload{}).Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var uploads []model.Upload
	if err := model.DB.Order("id DESC").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&uploads).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, pq.Data(uploads, total))
}

// AdminDeleteUpload DELETE /api/v1/admin/uploads/:id —— 同时删除磁盘文件（尽力而为）
func AdminDeleteUpload(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var upload model.Upload
	if err := model.DB.First(&upload, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "上传记录不存在")
		return
	}

	if upload.Path != "" {
		if p := safeUploadPath(uploadDir, upload.Path); p != "" {
			_ = os.Remove(p)
		}
	}
	if err := model.DB.Delete(&upload).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// safeUploadPath 防路径穿越：仅允许相对 uploadDir 的常规路径
func safeUploadPath(root, rel string) string {
	if strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") {
		return ""
	}
	clean := filepath.Clean(filepath.Join(root, rel))
	if !strings.HasPrefix(clean, filepath.Clean(root)+string(os.PathSeparator)) {
		return ""
	}
	return clean
}
