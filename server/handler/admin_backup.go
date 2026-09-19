package handler

// 备份管理（契约 #77-80）。安全要点：
// - 备份目录独立于 uploads，不经静态服务暴露；文件名只从服务端生成
// - 下载/删除按 name 白名单校验，防路径穿越
// - MySQL 模式下无 mysqldump 可用时明确报错，不产出假备份
// - 运行中数据库的热恢复不在 API 内执行（详见 database.md 说明）

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"myblog/server/common"
	"myblog/server/config"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// appCfg 运行配置（SetConfig 注入），备份需要 DBType/DBPath/UploadDir
var appCfg *config.Config

var backupNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func backupDir() string {
	dir := os.Getenv("BLOG_BACKUP_DIR")
	if dir == "" {
		dir = "backups"
	}
	return dir
}

// ensureBackupDir 创建备份目录（幂等）
func ensureBackupDir() error {
	return os.MkdirAll(backupDir(), 0o755)
}

// backupFileName 备份文件名：myblog-{database|full}-20060102-150405.{db|zip}
func backupFileName(kind string) string {
	ext := "zip"
	if kind == "database" {
		if appCfg.DBType == "sqlite" {
			ext = "db"
		}
	} else {
		kind = "full"
	}
	return fmt.Sprintf("myblog-%s-%s.%s", kind, time.Now().Format("20060102-150405"), ext)
}

// AdminListBackups GET /api/v1/admin/backups
func AdminListBackups(c *gin.Context) {
	dir := backupDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		common.OK(c, gin.H{"list": []gin.H{}})
		return
	}
	type backupItem struct {
		Name      string    `json:"name"`
		Size      int64     `json:"size"`
		CreatedAt time.Time `json:"createdAt"`
		Type      string    `json:"type"`
	}
	list := make([]backupItem, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !backupNamePattern.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		kind := "full"
		if strings.Contains(e.Name(), "-database-") {
			kind = "database"
		}
		list = append(list, backupItem{
			Name:      e.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			Type:      kind,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		if !list[i].CreatedAt.Equal(list[j].CreatedAt) {
			return list[i].CreatedAt.After(list[j].CreatedAt)
		}
		return list[i].Name < list[j].Name
	})
	common.OK(c, gin.H{"list": list})
}

// AdminCreateBackup POST /api/v1/admin/backups —— {type:"database"|"full"}
func AdminCreateBackup(c *gin.Context) {
	var req struct {
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Type != "database" && req.Type != "full") {
		common.Fail(c, common.CodeParamError, "type 仅支持 database / full")
		return
	}
	if err := ensureBackupDir(); err != nil {
		common.ServerError(c, err)
		return
	}

	var name string
	switch req.Type {
	case "database":
		if appCfg.DBType != "sqlite" {
			common.Fail(c, common.CodeParamError,
				"MySQL 模式不支持数据库文件备份，请使用全站导出（含 JSON 数据）")
			return
		}
		name = backupFileName("database")
		if err := copyFile(appCfg.DBPath, filepath.Join(backupDir(), name)); err != nil {
			notifyBackup(false, err.Error())
			common.ServerError(c, err)
			return
		}
	case "full":
		name = backupFileName("full")
		if err := createFullBackup(filepath.Join(backupDir(), name)); err != nil {
			notifyBackup(false, err.Error())
			common.ServerError(c, err)
			return
		}
	}

	info, err := os.Stat(filepath.Join(backupDir(), name))
	size := int64(0)
	modTime := time.Now()
	if err == nil {
		size, modTime = info.Size(), info.ModTime()
	}
	model.CreateAuditLog(model.DB, "backup.create", "backup", name, "创建"+req.Type+"备份", model.IPHash(c.ClientIP()))
	model.CreateNotification(model.DB, "backup", "备份完成",
		fmt.Sprintf("%s（%s）已创建", name, req.Type), "/backups")
	common.OK(c, gin.H{"item": gin.H{
		"name": name, "size": size, "createdAt": modTime, "type": req.Type,
	}})
}

// notifyBackup 备份失败通知
func notifyBackup(ok bool, message string) {
	if ok {
		_ = model.CreateNotification(model.DB, "backup", "备份完成", message, "/backups")
		return
	}
	_ = model.CreateNotification(model.DB, "backup", "备份失败", message, "/backups")
}

// createFullBackup 数据库文件 + uploads 目录打包 zip
func createFullBackup(dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)

	// 数据库（仅 SQLite 直接拷文件；MySQL 模式导出 JSON 结构由导出接口承担）
	if appCfg.DBType == "sqlite" {
		if err := addFileToZip(zw, appCfg.DBPath, "database/blog.db"); err != nil {
			return err
		}
	}
	// uploads 目录
	entries, err := os.ReadDir(appCfg.UploadDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if err := addFileToZip(zw, filepath.Join(appCfg.UploadDir, e.Name()), "media/"+e.Name()); err != nil {
				return err
			}
		}
	}
	return zw.Close()
}

func addFileToZip(zw *zip.Writer, src, name string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, srcFile)
	return err
}

// AdminDownloadBackup GET /api/v1/admin/backups/:name/download
func AdminDownloadBackup(c *gin.Context) {
	name := c.Param("name")
	if !backupNamePattern.MatchString(name) || strings.Contains(name, "..") {
		common.Fail(c, common.CodeParamError, "备份文件名不合法")
		return
	}
	path := filepath.Join(backupDir(), name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		common.Fail(c, common.CodeNotFound, "备份不存在")
		return
	}
	model.CreateAuditLog(model.DB, "backup.download", "backup", name, "下载备份", model.IPHash(c.ClientIP()))
	c.FileAttachment(path, name)
}

// AdminDeleteBackup DELETE /api/v1/admin/backups/:name
func AdminDeleteBackup(c *gin.Context) {
	name := c.Param("name")
	if !backupNamePattern.MatchString(name) || strings.Contains(name, "..") {
		common.Fail(c, common.CodeParamError, "备份文件名不合法")
		return
	}
	path := filepath.Join(backupDir(), name)
	if _, err := os.Stat(path); err != nil {
		common.Fail(c, common.CodeNotFound, "备份不存在")
		return
	}
	if err := os.Remove(path); err != nil {
		common.ServerError(c, err)
		return
	}
	model.CreateAuditLog(model.DB, "backup.delete", "backup", name, "删除备份", model.IPHash(c.ClientIP()))
	common.OK(c, nil)
}

// copyFile 简单文件复制
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
