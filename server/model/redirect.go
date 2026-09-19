package model

// Redirect URL 重定向（契约：站内路径 301/302，写入前做环检测）

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	RedirectPermanent = 301
	RedirectTemporary = 302
)

// RedirectLoopMaxDepth 沿 target 链回溯的最大层数，超过按环处理
const RedirectLoopMaxDepth = 10

type Redirect struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Source string `gorm:"type:varchar(512);uniqueIndex" json:"source"` // 旧站内路径，如 /post/go-guide
	Target string `gorm:"type:varchar(512)" json:"target"`             // 新站内路径
	Type   int    `gorm:"default:301" json:"type"`
	// Enabled 不加 default 标签：GORM 会对带默认值的零值字段做插入省略，false 会被写成 true
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NormalizeRedirectPath 规范化站内路径：去空白、去 query/hash、补前导 /；非法返回空串
func NormalizeRedirectPath(raw string) string {
	p := strings.TrimSpace(raw)
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// 拒绝纯 "/default?" 之类无法落地的路径
	if len(p) > 512 || strings.HasSuffix(p, "/.") || strings.Contains(p, "..") {
		return ""
	}
	return p
}

// RedirectLoopExists 沿 target 链回溯检测是否会回到 source（超最大层数视为环）
func RedirectLoopExists(db *gorm.DB, source, target string) bool {
	cur := target
	for i := 0; i < RedirectLoopMaxDepth; i++ {
		if cur == source {
			return true
		}
		var rule Redirect
		err := db.Where("source = ?", cur).First(&rule).Error
		if err != nil {
			return false // 链终止
		}
		cur = rule.Target
	}
	return true
}

// UpsertRedirectForSlug 按来源路径 upsert 重定向（slug 变更自动 301 用）
func UpsertRedirectForSlug(db *gorm.DB, source, target string) error {
	var rule Redirect
	err := db.Where("source = ?", source).First(&rule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rule = Redirect{Source: source, Target: target, Type: RedirectPermanent, Enabled: true}
		return db.Create(&rule).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&rule).Updates(map[string]any{"target": target, "type": RedirectPermanent, "enabled": true}).Error
}
