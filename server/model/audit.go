package model

// AuditLog 操作日志（契约 #76）。安全边界：禁止记录密码/JWT/Authorization/完整请求体，
// description 仅存截断摘要。

import (
	"time"

	"gorm.io/gorm"
)

type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Action       string    `gorm:"index;type:varchar(50)" json:"action"` // post.create / comment.batch / backup.create ...
	ResourceType string    `gorm:"type:varchar(50)" json:"resourceType"`
	ResourceID   string    `gorm:"type:varchar(50)" json:"resourceId"`
	Description  string    `gorm:"type:varchar(500)" json:"description"`
	IPHash       string    `gorm:"type:varchar(64)" json:"ipHash"`
	CreatedAt    time.Time `gorm:"index" json:"createdAt"`
}

// IPHash 管理员/访客 IP 的哈希（审计日志用，不存明文）
func IPHash(ip string) string {
	return hashWithSalt(ip)
}

// CreateAuditLog 写入操作日志；description 超 500 字截断。
// 日志失败不影响业务（返回错误由调用方决定忽略）。
func CreateAuditLog(db *gorm.DB, action, resourceType, resourceID, description, ipHash string) error {
	log := AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Description:  description,
		IPHash:       ipHash,
	}
	if r := []rune(log.Description); len(r) > 500 {
		log.Description = string(r[:500])
	}
	return db.Create(&log).Error
}
