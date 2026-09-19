// Package config 负责加载环境变量配置，全部有默认值，零配置可跑（SQLite）。
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config 服务运行配置
type Config struct {
	Port      string // 监听端口
	JWTSecret string // JWT 签名密钥
	DBType    string // sqlite | mysql
	DBPath    string // SQLite 文件路径
	MySQLDSN  string // MySQL DSN（BLOG_DB_TYPE=mysql 时必填）
	UploadDir string // 上传文件存储目录（/uploads 静态服务根）

	// 定时发布
	ScheduleInterval time.Duration // 到期定时文章的扫描间隔（BLOG_SCHEDULE_INTERVAL_SECONDS，默认 30s）

	// 安全配置
	CryptoKey      string   // 敏感字段加密密钥（64 位 hex = 32 字节 AES-256），见 model/crypto.go
	TrustedProxies []string // 可信代理（IP/CIDR）；ClientIP 仅从这些代理解析 X-Forwarded-For
	CORSOrigins    []string // CORS Origin 白名单；空 = 放开所有 Origin（历史行为）

	// 防护配置（反爬 / 恶意请求拦截 / 自动封禁），见 middleware/defense.go
	RateLimitPerMin int           // 全局每 IP 每分钟请求数上限（0=关闭，默认 240）
	WAFMode         string        // 恶意请求拦截模式：off | log | block（默认 block）
	BanThreshold    int           // 滑动窗口内违规计点达到阈值自动封禁（0=关闭自动封禁，默认 10）
	BanWindow       time.Duration // 违规计点滑动窗口（默认 10 分钟）
	BanDuration     time.Duration // 自动封禁时长（默认 30 分钟）
	BanWhitelist    []string      // 防护白名单 IP/CIDR：完全跳过封禁/WAF/限流（默认回环）
}

// Load 读取环境变量并校验
func Load() *Config {
	cfg := &Config{
		Port:        getEnv("BLOG_PORT", "8080"),
		JWTSecret:   strings.TrimSpace(getEnv("BLOG_JWT_SECRET", "")),
		DBType:      strings.ToLower(strings.TrimSpace(getEnv("BLOG_DB_TYPE", "sqlite"))),
		DBPath:      getEnv("BLOG_DB_PATH", "blog.db"),
		MySQLDSN:    strings.TrimSpace(getEnv("BLOG_MYSQL_DSN", "")),
		UploadDir:   getEnv("BLOG_UPLOAD_DIR", "uploads"),
		CORSOrigins: splitList(getEnv("BLOG_CORS_ORIGINS", "")),
		TrustedProxies: splitList(getEnv("BLOG_TRUSTED_PROXIES",
			"127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")),
	}

	// 定时发布扫描间隔：非法值回退默认，下限 5s 防止误配打满数据库
	scheduleSeconds, err := strconv.Atoi(getEnv("BLOG_SCHEDULE_INTERVAL_SECONDS", "30"))
	if err != nil || scheduleSeconds < 5 {
		scheduleSeconds = 30
	}
	cfg.ScheduleInterval = time.Duration(scheduleSeconds) * time.Second

	// 防护配置：限流 0=关闭；WAF 模式枚举校验；时长下限 1 分钟防误配
	cfg.RateLimitPerMin = getInt("BLOG_RATE_LIMIT_PER_MIN", 240)
	cfg.WAFMode = strings.ToLower(getEnv("BLOG_WAF_MODE", "block"))
	switch cfg.WAFMode {
	case "off", "log", "block":
	default:
		log.Fatalf("config: BLOG_WAF_MODE 仅支持 off / log / block，当前为 %q", cfg.WAFMode)
	}
	cfg.BanThreshold = getInt("BLOG_BAN_THRESHOLD", 10)
	banWindowMinutes := getInt("BLOG_BAN_WINDOW_MINUTES", 10)
	if banWindowMinutes < 1 {
		banWindowMinutes = 1
	}
	cfg.BanWindow = time.Duration(banWindowMinutes) * time.Minute
	banDurationMinutes := getInt("BLOG_BAN_DURATION_MINUTES", 30)
	if banDurationMinutes < 1 {
		banDurationMinutes = 1
	}
	cfg.BanDuration = time.Duration(banDurationMinutes) * time.Minute
	// 白名单支持显式置空（关掉默认回环豁免）：必须用 LookupEnv 区分「未设置」与「设为空」
	if raw, ok := os.LookupEnv("BLOG_BAN_WHITELIST"); ok {
		cfg.BanWhitelist = splitList(raw)
	} else {
		cfg.BanWhitelist = splitList("127.0.0.1,::1")
	}

	// JWT 密钥：未配置时自动生成并落盘（与 SQLite 同目录，容器部署时随卷持久化），
	// 重启不失效；禁止回退到已知的硬编码默认值。
	if cfg.JWTSecret == "" {
		secretFile := secretFilePath(cfg, "blog.jwt.secret")
		secret, err := loadOrCreateSecretFile(secretFile, 32)
		if err != nil {
			log.Fatalf("config: 读取/生成 JWT 密钥文件 %s 失败：%v", secretFile, err)
		}
		cfg.JWTSecret = secret
		log.Printf("config: 未设置 BLOG_JWT_SECRET，已自动生成密钥文件 %s（多实例部署请显式配置环境变量）", secretFile)
	}

	// 敏感字段加密密钥：BLOG_CRYPTO_KEY（openssl rand -hex 32）或自动生成落盘。
	// 密钥丢失后已加密数据（评论邮箱/IP 等）无法解密，务必随数据库一起备份。
	if keyEnv := strings.TrimSpace(getEnv("BLOG_CRYPTO_KEY", "")); keyEnv != "" {
		if len(keyEnv) != 64 {
			log.Fatal("config: BLOG_CRYPTO_KEY 必须是 64 位 hex（32 字节，可用 openssl rand -hex 32 生成）")
		}
		if _, err := hex.DecodeString(keyEnv); err != nil {
			log.Fatalf("config: BLOG_CRYPTO_KEY 不是合法 hex：%v", err)
		}
		cfg.CryptoKey = keyEnv
	} else {
		keyFile := secretFilePath(cfg, "blog.crypto.key")
		key, err := loadOrCreateSecretFile(keyFile, 32)
		if err != nil {
			log.Fatalf("config: 读取/生成加密密钥文件 %s 失败：%v", keyFile, err)
		}
		cfg.CryptoKey = key
		log.Printf("config: 敏感字段加密密钥文件 %s（勿删；删除或更换后已加密数据无法解密）", keyFile)
	}

	// 可信代理必须是合法 IP / CIDR，错误尽早暴露
	for _, p := range cfg.TrustedProxies {
		if net.ParseIP(p) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(p); err != nil {
			log.Fatalf("config: BLOG_TRUSTED_PROXIES 含非法 IP/CIDR %q", p)
		}
	}

	// 防护白名单同理必须是合法 IP / CIDR
	for _, p := range cfg.BanWhitelist {
		if net.ParseIP(p) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(p); err != nil {
			log.Fatalf("config: BLOG_BAN_WHITELIST 含非法 IP/CIDR %q", p)
		}
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

// getInt 解析非负整数环境变量；未设置/非法回退默认值
func getInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
		log.Printf("config: %s=%q 不是合法的非负整数，回退默认值 %d", key, v, def)
	}
	return def
}

// splitList 解析逗号分隔列表，忽略空白项
func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// secretFilePath 密钥文件默认与 SQLite 数据库同目录
// （容器内 DB_PATH=/app/data/blog.db，该目录挂卷持久化；裸机 SQLite 即数据所在目录）
func secretFilePath(cfg *Config, name string) string {
	if dir := filepath.Dir(cfg.DBPath); dir != "" && dir != "." {
		return filepath.Join(dir, name)
	}
	return name
}

// loadOrCreateSecretFile 读取 hex 密钥文件；不存在则生成 nBytes 字节随机数并以 0600 写入
func loadOrCreateSecretFile(path string, nBytes int) (string, error) {
	if v, err := readSecretFile(path); err == nil && v != "" {
		return v, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成随机密钥: %w", err)
	}
	hexStr := hex.EncodeToString(buf)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) { // 并发启动竞争：文件已被创建，直接读取
		v, rerr := readSecretFile(path)
		if rerr != nil || v == "" {
			return "", fmt.Errorf("密钥文件 %s 已存在但不可读或为空", path)
		}
		return v, nil
	}
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(hexStr); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return hexStr, nil
}

func readSecretFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
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
