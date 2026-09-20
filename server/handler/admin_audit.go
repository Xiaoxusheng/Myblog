package handler

// 操作日志（契约 #76）：列表查询 + 各业务 handler 的写点辅助。

import (
	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// writeAudit 记录一条操作日志（ip 哈希化）；失败静默，不阻断业务
func writeAudit(c *gin.Context, action, resourceType, resourceID, description string) {
	_ = model.CreateAuditLog(model.DB, action, resourceType, resourceID, description, model.IPHash(c.ClientIP()))
}

// AdminListAuditLogs GET /api/v1/admin/audit-logs?page=&pageSize=&action=
func AdminListAuditLogs(c *gin.Context) {
	pq := common.ParsePage(c, 20)
	actionPrefix := c.Query("action")

	buildQuery := func() *gorm.DB {
		db := model.DB.Model(&model.AuditLog{})
		if actionPrefix != "" {
			// 前缀过滤：转义用户输入中的 LIKE 通配符，防通配符注入
			db = db.Where("action "+likeEscapeClause, escapeLike(actionPrefix)+"%")
		}
		return db
	}
	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var list []model.AuditLog
	if err := buildQuery().Order("id DESC").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&list).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, pq.Data(list, total))
}
