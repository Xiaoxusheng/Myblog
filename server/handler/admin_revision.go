package handler

// 管理端文章版本历史（契约 #50-52）。
// 版本由保存/恢复流程自动生成（仅内容真正变化）；本文件只读列表/详情与执行恢复。

import (
	"fmt"
	"time"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminListRevisions GET /api/v1/admin/posts/:id/revisions —— version 倒序分页
func AdminListRevisions(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "文章 id 不合法")
		return
	}
	var post model.Post
	if err := model.DB.Select("id").First(&post, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}

	pq := common.ParsePage(c, 10)
	buildQuery := func() *gorm.DB {
		return model.DB.Model(&model.PostRevision{}).
			Where("post_id = ?", id).Order("version DESC")
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var revs []model.PostRevision
	if err := buildQuery().Offset(pq.Offset()).Limit(pq.PageSize).Find(&revs).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	// 列表项不含 content（契约 PostRevisionItem）
	type revisionItem struct {
		ID        uint      `json:"id"`
		PostID    uint      `json:"postId"`
		Version   int       `json:"version"`
		Remark    string    `json:"remark"`
		CreatedAt time.Time `json:"createdAt"`
	}
	list := make([]revisionItem, 0, len(revs))
	for i := range revs {
		list = append(list, revisionItem{
			ID:        revs[i].ID,
			PostID:    revs[i].PostID,
			Version:   revs[i].Version,
			Remark:    revs[i].Remark,
			CreatedAt: revs[i].CreatedAt,
		})
	}
	common.OK(c, pq.Data(list, total))
}

// AdminGetRevision GET /api/v1/admin/posts/:id/revisions/:version —— 全量快照
func AdminGetRevision(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "文章 id 不合法")
		return
	}
	version, err := strconvUint(c.Param("version"))
	if err != nil || version == 0 {
		common.Fail(c, common.CodeParamError, "版本号不合法")
		return
	}
	var rev model.PostRevision
	if err := model.DB.Where("post_id = ? AND version = ?", id, version).First(&rev).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "版本不存在")
		return
	}
	common.OK(c, gin.H{"revision": rev})
}

// AdminRestoreRevision POST /api/v1/admin/posts/:id/revisions/:version/restore
// 恢复语义（契约 #52）：当前内容先快照（恢复前快照）→ 应用目标版本内容字段 → 生成「恢复自 vN」。
// 仅应用 title/slug/summary/cover/content/category_id/is_top，不改当前发布状态与计划时间；
// 无实际变化不生成版本；目标 Slug 已被占用 → 10001 回滚。
func AdminRestoreRevision(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "文章 id 不合法")
		return
	}
	version, err := strconvUint(c.Param("version"))
	if err != nil || version == 0 {
		common.Fail(c, common.CodeParamError, "版本号不合法")
		return
	}

	var post model.Post
	if err := model.DB.First(&post, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}
	var target model.PostRevision
	if err := model.DB.Where("post_id = ? AND version = ?", id, version).First(&target).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "版本不存在")
		return
	}

	before := model.SnapshotOf(&post)
	after := before
	after.Title = target.Title
	after.Slug = target.Slug
	after.Summary = target.Summary
	after.Cover = target.Cover
	after.Content = target.Content
	after.CategoryID = target.CategoryID
	after.IsTop = target.IsTop

	// 内容无实际变化：不生成版本，直接返回当前文章
	if model.SnapshotEqual(before, after) {
		respondAdminPost(c, id)
		return
	}
	// 恢复会改写 slug，先做唯一性预检，避免事务中途撞 uniqueIndex
	if target.Slug != post.Slug && slugTaken(model.DB, target.Slug, post.ID) {
		common.Fail(c, common.CodeParamError, "恢复的 Slug 已被其他文章占用，请先修改当前 Slug")
		return
	}

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		// 当前内容与最新版本不同 → 先快照，保证恢复可撤销
		latest, err := model.LatestPostRevision(tx, post.ID)
		if err != nil {
			return err
		}
		if latest == nil || !model.SnapshotEqual(before, latest.Snapshot()) {
			if err := model.CreatePostRevision(tx, &post, "恢复前快照"); err != nil {
				return err
			}
		}
		updates := map[string]any{
			"title":       target.Title,
			"slug":        target.Slug,
			"summary":     target.Summary,
			"cover":       target.Cover,
			"content":     target.Content,
			"category_id": target.CategoryID,
			"is_top":      target.IsTop,
		}
		if err := tx.Model(&post).Updates(updates).Error; err != nil {
			return err
		}
		// Updates(map) 不回写结构体，手动同步供版本快照
		post.Title = target.Title
		post.Slug = target.Slug
		post.Summary = target.Summary
		post.Cover = target.Cover
		post.Content = target.Content
		post.CategoryID = target.CategoryID
		post.IsTop = target.IsTop
		return model.CreatePostRevision(tx, &post, fmt.Sprintf("恢复自 v%d", target.Version))
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	writeAudit(c, "post.restore", "post", c.Param("id"), fmt.Sprintf("恢复自 v%d", version))
	respondAdminPost(c, id)
}

// respondAdminPost 重新加载文章（含关联）并按契约返回 {post:AdminPostItem}
func respondAdminPost(c *gin.Context, id uint) {
	var post model.Post
	if err := model.DB.Preload("Category").Preload("Tags").First(&post, id).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	counts := commentCountsByPost([]uint{post.ID})
	common.OK(c, gin.H{"post": model.ToAdminPostItem(&post, counts[post.ID])})
}
