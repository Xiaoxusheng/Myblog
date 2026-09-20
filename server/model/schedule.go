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
//
// 注意：比较基准必须传 UTC。glebarez/sqlite 把 time.Time 落库为带 Z 的 RFC3339
// 字符串（如 "2026-09-20T06:07:31Z"），而 SQLite 的比较是**字典序字符串比较**；
// 若绑定本地时间（"2026-09-20 11:07:31+08:00"），第 11 位空格(0x20) < 'T'(0x54)，
// 会让「当天未来的计划」排到「现在」之前 → 未到点立刻被误发布。
// 绑定 UTC 后两侧格式同构，字典序才等价于时间先后。
func PublishDueScheduledPosts(db *gorm.DB) (int64, error) {
	// 单条 UPDATE：SET 均取更新前的行值，published_at 写入原计划时间、publish_at 置空
	res := db.Model(&Post{}).
		Where("status = ? AND publish_at IS NOT NULL AND publish_at <= ?", PostScheduled, time.Now().UTC()).
		Updates(map[string]any{
			"status":       PostPublished,
			"published_at": gorm.Expr("publish_at"),
			"publish_at":   nil,
		})
	return res.RowsAffected, res.Error
}

// PublishDueScheduledPages 将计划时间已到的定时发布页面置为已发布：
// 语义与 PublishDueScheduledPosts 完全一致（status 3→1、published_at 取计划时间、清空 publish_at）。
// 由 main 的调度器在启动补跑与轮询中与文章一同调用；系统行为，不生成版本。
// 比较基准同样必须传 UTC，原因见 PublishDueScheduledPosts。
func PublishDueScheduledPages(db *gorm.DB) (int64, error) {
	res := db.Model(&Page{}).
		Where("status = ? AND publish_at IS NOT NULL AND publish_at <= ?", PostScheduled, time.Now().UTC()).
		Updates(map[string]any{
			"status":       PostPublished,
			"published_at": gorm.Expr("publish_at"),
			"publish_at":   nil,
		})
	return res.RowsAffected, res.Error
}
