package model

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"myblog/server/config"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库句柄（博客规模单库，handler 直接使用）
var DB *gorm.DB

// Open 按配置打开数据库连接；成功后将句柄写入全局 DB
func Open(cfg *config.Config) (*gorm.DB, error) {
	// 敏感字段加密密钥（hex 64 → 32 字节）；空串关闭加密（未启用场景）
	var key []byte
	if cfg.CryptoKey != "" {
		decoded, err := hex.DecodeString(cfg.CryptoKey)
		if err != nil || len(decoded) != 32 {
			return nil, errors.New("BLOG_CRYPTO_KEY 必须是 64 位 hex（32 字节）")
		}
		key = decoded
	}
	if err := SetCryptoKey(key); err != nil {
		return nil, fmt.Errorf("初始化敏感字段加密: %w", err)
	}

	var dialector gorm.Dialector
	switch cfg.DBType {
	case "mysql":
		if cfg.MySQLDSN == "" {
			return nil, errors.New("BLOG_DB_TYPE=mysql 需要设置 BLOG_MYSQL_DSN 环境变量")
		}
		dialector = mysql.Open(cfg.MySQLDSN)
	default:
		dialector = sqlite.Open(cfg.DBPath)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if cfg.DBType == "mysql" {
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(time.Hour)
	} else {
		// SQLite：单连接串行化读写，规避 "database is locked"；
		// 内存库（测试）也依赖单连接保活，避免表随连接销毁。
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}

	DB = db
	return db, nil
}

// AutoMigrate 建表/补列（幂等）
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Post{},
		&PostRevision{},
		&Category{},
		&Tag{},
		&PostTag{},
		&Series{},
		&Comment{},
		&Link{},
		&Page{},
		&Setting{},
		&Upload{},
		&Redirect{},
		&PageView{},
		&CommentBlacklist{},
		&Notification{},
		&AuditLog{},
		&SearchLog{},
	)
}

// mustLog seed 过程中的非致命错误只记录日志，不中断启动
func mustLog(scope string, err error) {
	if err != nil {
		log.Printf("[seed] %s: %v", scope, err)
	}
}
