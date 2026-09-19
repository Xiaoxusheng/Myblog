package model

// 定时发布调度（契约：status=3 到点由后端自动置为已发布，重启不丢计划——数据在库，补跑即可）。

import (
	"time"

	"gorm.io/gorm"
)

// PublishDueScheduledPosts 将计划时间已到的定时发布文章置为已发布：
// status 3→1，published_at 取计划时间 publish_at（前台时间线与计划一致），
// 并按契约清空 publish_at（仅定时发布状态携带计划时间）。
// 返回发布篇数。由 main 的调度器在启动补跑与轮询中调用；系统行为，不生成版本。
func PublishDueScheduledPosts(db *gorm.DB) (int64, error) {
	// 单条 UPDATE：SET 均取更新前的行值，published_at 写入原计划时间、publish_at 置空
	res := db.Model(&Post{}).
		Where("status = ? AND publish_at IS NOT NULL AND publish_at <= ?", PostScheduled, time.Now()).
		Updates(map[string]any{
			"status":       PostPublished,
			"published_at": gorm.Expr("publish_at"),
			"publish_at":   nil,
		})
	return res.RowsAffected, res.Error
}
