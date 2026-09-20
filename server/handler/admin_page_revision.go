package handler

// 管理端页面版本历史（契约 #101-103）。
// 版本由保存/恢复流程自动生成（仅内容真正变化）；本文件只读列表/详情与执行恢复。

import (
	"fmt"
	"time"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminListPageRevisions GET /api/v1/admin/pages/:id/revisions —— version 倒序分页
func AdminListPageRevisions(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var page model.Page
	if err := model.DB.Select("id").First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}

	pq := common.ParsePage(c, 10)
	buildQuery := func() *gorm.DB {
		return model.DB.Model(&model.PageRevision{}).
			Where("page_id = ?", id).Order("version DESC")
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var revs []model.PageRevision
	if err := buildQuery().Offset(pq.Offset()).Limit(pq.PageSize).Find(&revs).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	// 列表项不含 content（契约 PageRevisionItem）
	list := make([]model.PageRevisionItemDTO, 0, len(revs))
	for i := range revs {
		list = append(list, model.ToPageRevisionItem(&revs[i]))
	}
	common.OK(c, pq.Data(list, total))
}

// AdminGetPageRevision GET /api/v1/admin/pages/:id/revisions/:version —— 全量快照
func AdminGetPageRevision(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	version, err := strconvUint(c.Param("version"))
	if err != nil || version == 0 {
		common.Fail(c, common.CodeParamError, "版本号不合法")
		return
	}
	var rev model.PageRevision
	if err := model.DB.Where("page_id = ? AND version = ?", id, version).First(&rev).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "版本不存在")
		return
	}
	common.OK(c, gin.H{"revision": model.ToPageRevisionDetail(&rev)})
}

// AdminRestorePageRevision POST /api/v1/admin/pages/:id/revisions/:version/restore
// 恢复语义（契约 #103）：当前内容先快照（恢复前快照）→ 应用目标版本内容字段 → 生成「恢复自 vN」。
// 仅应用 title/slug/content，不改当前发布状态与计划时间；
// 无实际变化不生成版本；目标 slug 已被占用 → 10001 回滚。
func AdminRestorePageRevision(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	version, err := strconvUint(c.Param("version"))
	if err != nil || version == 0 {
		common.Fail(c, common.CodeParamError, "版本号不合法")
		return
	}

	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}
	var target model.PageRevision
	if err := model.DB.Where("page_id = ? AND version = ?", id, version).First(&target).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "版本不存在")
		return
	}

	before := model.PageSnapshotOf(&page)
	after := before
	after.Title = target.Title
	after.Slug = target.Slug
	after.Content = target.Content

	// 内容无实际变化：不生成版本，直接返回当前页面
	if model.PageSnapshotEqual(before, after) {
		respondAdminPage(c, id)
		return
	}
	// 恢复会改写 slug，先做唯一性预检，避免事务中途撞 uniqueIndex
	if target.Slug != page.Slug && pageSlugTaken(target.Slug, page.ID) {
		common.Fail(c, common.CodeParamError, "恢复的 Slug 已被其他页面占用，请先修改当前 Slug")
		return
	}

	now := normalizeMillis(time.Now())
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		// 当前内容与最新版本不同 → 先快照，保证恢复可撤销
		latest, err := model.LatestPageRevision(tx, page.ID)
		if err != nil {
			return err
		}
		if latest == nil || !model.PageSnapshotEqual(before, latest.Snapshot()) {
			if err := model.CreatePageRevision(tx, &page, "恢复前快照"); err != nil {
				return err
			}
		}
		updates := map[string]any{
			"title":      target.Title,
			"slug":       target.Slug,
			"content":    target.Content,
			"updated_at": now,
		}
		if err := tx.Model(&page).Omit("updated_at").Updates(updates).Error; err != nil {
			return err
		}
		// Updates(map) 不回写结构体，手动同步供版本快照
		page.Title = target.Title
		page.Slug = target.Slug
		page.Content = target.Content
		page.UpdatedAt = now
		if err := tx.Model(&page).UpdateColumn("updated_at", now).Error; err != nil {
			return err
		}
		return model.CreatePageRevision(tx, &page, fmt.Sprintf("恢复自 v%d", target.Version))
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	writeAudit(c, "page.restore", "page", c.Param("id"), fmt.Sprintf("恢复自 v%d", version))
	respondAdminPage(c, id)
}
