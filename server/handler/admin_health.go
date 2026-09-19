package handler

// 系统健康状态与版本（契约 #84，Phase 7）。
// 采集范围刻意克制：版本/Go 版本/数据库类型与连通性/内容计数/上传目录大小/最近备份；
// 磁盘空间等强平台相关项不做（避免 CGO/平台差异），见部署文档说明。

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// serverVersion 构建时可注入（-ldflags "-X myblog/server/handler.serverVersion=v1.x"）
var serverVersion = "dev"

// SetVersion 由 main 注入版本号
func SetVersion(v string) {
	if v != "" {
		serverVersion = v
	}
}

// Health GET /api/v1/admin/health
func Health(c *gin.Context) {
	db := model.DB

	// 数据库连通性
	dbErr := db.Exec("SELECT 1").Error
	dbStatus := "ok"
	if dbErr != nil {
		dbStatus = "error: " + dbErr.Error()
	}

	// 内容计数
	var postCount, commentCount, mediaCount int64
	_ = db.Model(&model.Post{}).Count(&postCount).Error
	_ = db.Model(&model.Comment{}).Count(&commentCount).Error
	_ = db.Model(&model.Upload{}).Count(&mediaCount).Error

	// 上传目录大小
	uploadSize := dirSize(appCfg.UploadDir)

	// 最近备份
	latestBackup := latestBackupAt(backupDir())

	dbType := appCfg.DBType
	common.OK(c, gin.H{
		"version":        serverVersion,
		"goVersion":      runtime.Version(),
		"dbType":         dbType,
		"dbStatus":       dbStatus,
		"postCount":      postCount,
		"commentCount":   commentCount,
		"mediaCount":     mediaCount,
		"uploadSize":     uploadSize,
		"latestBackupAt": latestBackup,
	})
}

// dirSize 递归统计目录大小（字节）
func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

// latestBackupAt 最近一次备份时间（扫描备份目录，取最新文件修改时间）
func latestBackupAt(dir string) *time.Time {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return nil
	}
	var latest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if info, err := e.Info(); err == nil && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	if latest.IsZero() {
		return nil
	}
	return &latest
}
