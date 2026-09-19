package model

// IP 封禁持久化（契约「安全防护」）。自动/手动封禁均落库，
// 进程重启后由 middleware.Guard 启动加载，封禁不因重启失效。
// 封禁 IP 存明文：与 comment_blacklist 的 ip 值一致，是运营必需数据，
// 且需精确等值匹配（page_views 的哈希方案无法反向匹配）。

import (
	"time"

	"gorm.io/gorm"
)

// BannedIP 被封禁的 IP
type BannedIP struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	IP        string     `gorm:"uniqueIndex;type:varchar(64)" json:"ip"`
	Reason    string     `gorm:"type:varchar(200)" json:"reason"`
	Source    string     `gorm:"type:varchar(16)" json:"source"` // auto（自动）/ manual（手动）
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt"` // nil = 永久
}

// UpsertBan 写入/更新封禁记录（同 IP 覆盖）。
// Assign 必须用 map：结构体更新会跳过零值，导致临时封禁无法被永久封禁（expires_at=nil）覆盖。
func UpsertBan(db *gorm.DB, ip, reason, source string, expiresAt *time.Time) error {
	ban := BannedIP{IP: ip, Reason: reason, Source: source, ExpiresAt: expiresAt}
	return db.Where("ip = ?", ip).
		Assign(map[string]any{"reason": reason, "source": source, "expires_at": expiresAt}).
		FirstOrCreate(&ban).Error
}

// DeleteBan 删除封禁记录
func DeleteBan(db *gorm.DB, ip string) error {
	return db.Where("ip = ?", ip).Delete(&BannedIP{}).Error
}

// LoadActiveBans 加载全部生效中的封禁，并顺手清理已过期记录（启动时调用）
func LoadActiveBans(db *gorm.DB) ([]BannedIP, error) {
	var list []BannedIP
	if err := db.Where("expires_at IS NULL OR expires_at > ?", time.Now()).Find(&list).Error; err != nil {
		return nil, err
	}
	if err := db.Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now()).Delete(&BannedIP{}).Error; err != nil {
		return list, err // 清理失败不影响已加载的名单
	}
	return list, nil
}
