// Package config 负责加载环境变量配置，全部有默认值，零配置可跑（SQLite）。
package config

import (
	"log"
	"os"
	"strings"
)

// Config 服务运行配置
type Config struct {
	Port      string // 监听端口
	JWTSecret string // JWT 签名密钥
	DBType    string // sqlite | mysql
	DBPath    string // SQLite 文件路径
	MySQLDSN  string // MySQL DSN（BLOG_DB_TYPE=mysql 时必填）
	UploadDir string // 上传文件存储目录（/uploads 静态服务根）
}

// Load 读取环境变量并校验
func Load() *Config {
	cfg := &Config{
		Port:      getEnv("BLOG_PORT", "8080"),
		JWTSecret: getEnv("BLOG_JWT_SECRET", "dev-secret"),
		DBType:    strings.ToLower(strings.TrimSpace(getEnv("BLOG_DB_TYPE", "sqlite"))),
		DBPath:    getEnv("BLOG_DB_PATH", "blog.db"),
		MySQLDSN:  strings.TrimSpace(getEnv("BLOG_MYSQL_DSN", "")),
		UploadDir: getEnv("BLOG_UPLOAD_DIR", "uploads"),
	}

	if cfg.DBType != "sqlite" && cfg.DBType != "mysql" {
		log.Fatalf("config: 不支持的 BLOG_DB_TYPE=%q，仅支持 sqlite / mysql", cfg.DBType)
	}
	if cfg.DBType == "mysql" {
		if cfg.MySQLDSN == "" {
			log.Fatal("config: BLOG_DB_TYPE=mysql 时必须设置 BLOG_MYSQL_DSN 环境变量")
		}
		cfg.MySQLDSN = normalizeMySQLDSN(cfg.MySQLDSN)
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// normalizeMySQLDSN 校验并补齐 DSN 必需参数：
// charset=utf8mb4、parseTime=True、loc=Local（缺省时自动追加并提示）。
func normalizeMySQLDSN(dsn string) string {
	type param struct {
		key   string
		value string
	}
	required := []param{
		{key: "charset=", value: "charset=utf8mb4"},
		{key: "parseTime=", value: "parseTime=True"},
		{key: "loc=", value: "loc=Local"},
	}

	var missing []string
	for _, p := range required {
		if !strings.Contains(dsn, p.key) {
			missing = append(missing, p.value)
		}
	}
	if len(missing) == 0 {
		return dsn
	}

	log.Printf("config: BLOG_MYSQL_DSN 缺少必需参数 %s，已自动追加（建议显式写入 DSN）", strings.Join(missing, "&"))
	if strings.Contains(dsn, "?") {
		dsn += "&"
	} else {
		dsn += "?"
	}
	return dsn + strings.Join(missing, "&")
}
