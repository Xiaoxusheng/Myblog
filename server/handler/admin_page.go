package handler

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------- 自定义页面（契约 #38-44 / #101-105） ----------

// pageStatusCounts 列表概览：按当前 keyword 过滤后的全量统计（不受分页影响）
type pageStatusCounts struct {
	Total     int64
	Published int64
	Draft     int64
	Scheduled int64
}

// AdminListPages GET /api/v1/admin/pages?keyword=&status=&page=&pageSize=
func AdminListPages(c *gin.Context) {
	pq := common.ParsePage(c, 10)
	keyword := strings.TrimSpace(c.Query("keyword"))

	base := model.DB.Model(&model.Page{})
	if keyword != "" {
		like := likeContains(keyword)
		base = base.Where("title "+likeEscapeClause+" OR slug "+likeEscapeClause, like, like)
	}

	var counts pageStatusCounts
	countQuery := base.Session(&gorm.Session{})
	if err := countQuery.Count(&counts.Total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	// 概览分状态计数（复用同一过滤条件）
	type row struct {
		Status int8
		N      int64
	}
	var rows []row
	if err := countQuery.Session(&gorm.Session{}).
		Select("status, COUNT(*) AS n").Group("status").Scan(&rows).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	for _, r := range rows {
		switch r.Status {
		case model.PostPublished:
			counts.Published = r.N
		case model.PostDraft:
			counts.Draft = r.N
		case model.PostScheduled:
			counts.Scheduled = r.N
		}
	}

	listQuery := base.Session(&gorm.Session{})
	if v := strings.TrimSpace(c.Query("status")); v != "" {
		listQuery = listQuery.Where("status = ?", v)
	}

	var total int64
	if err := listQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var pages []model.Page
	if err := listQuery.Order("updated_at DESC, id DESC").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&pages).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	items := make([]model.PageItemDTO, 0, len(pages))
	for i := range pages {
		items = append(items, model.ToPageItem(&pages[i]))
	}

	data := pq.Data(items, total)
	common.OK(c, gin.H{
		"list":     data["list"],
		"total":    data["total"],
		"page":     data["page"],
		"pageSize": data["pageSize"],
		"meta": model.PageMetaDTO{
			TotalCount:     counts.Total,
			PublishedCount: counts.Published,
			DraftCount:     counts.Draft,
			ScheduledCount: counts.Scheduled,
		},
	})
}

// pagePayload 创建/更新页面入参
type pagePayload struct {
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
	Status  int8   `json:"status"`
	// PageType 页面类型（缺省 default）
	PageType string `json:"pageType"`
	// PublishAt 仅 status=3 需要
	PublishAt *time.Time `json:"publishAt"`
	// Auto 前端自动保存标记：距最新版本 <RevisionAutoThrottle 时不生成版本
	Auto bool `json:"auto"`
	// BaseUpdatedAt 并发编辑基线（编辑器读取时的 page.updatedAt）
	BaseUpdatedAt *time.Time `json:"baseUpdatedAt"`
	// SEO 四字段
	SeoTitle       string `json:"seoTitle"`
	SeoDescription string `json:"seoDescription"`
	Canonical      string `json:"canonical"`
	OgImage        string `json:"ogImage"`
}

// normalizePageType 页面类型缺省为 default
func normalizePageType(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return model.PageTypeDefault
	}
	return t
}

func (p *pagePayload) validate(c *gin.Context) bool {
	title := strings.TrimSpace(p.Title)
	switch {
	case title == "":
		common.Fail(c, common.CodeParamError, "标题不能为空")
	case utf8.RuneCountInString(title) > 200:
		common.Fail(c, common.CodeParamError, "标题不能超过 200 字")
	case utf8.RuneCountInString(strings.TrimSpace(p.Slug)) > 200:
		common.Fail(c, common.CodeParamError, "slug 不能超过 200 字符")
	case p.Status < model.PostDraft || p.Status > model.PostScheduled:
		common.Fail(c, common.CodeParamError, "status 仅允许 0（草稿）/ 1（已发布）/ 2（隐藏）/ 3（定时发布）")
	case p.Status == model.PostScheduled && p.PublishAt == nil:
		common.Fail(c, common.CodeParamError, "定时发布必须提供 publishAt")
	case !model.PageTypeValid(normalizePageType(p.PageType)):
		common.Fail(c, common.CodeParamError, "pageType 仅允许 default / about / links / contact")
	case utf8.RuneCountInString(p.SeoTitle) > 200:
		common.Fail(c, common.CodeParamError, "SEO 标题不能超过 200 字")
	case utf8.RuneCountInString(p.SeoDescription) > 300:
		common.Fail(c, common.CodeParamError, "SEO 描述不能超过 300 字")
	case utf8.RuneCountInString(p.Canonical) > 512:
		common.Fail(c, common.CodeParamError, "Canonical 不能超过 512 字符")
	case utf8.RuneCountInString(p.OgImage) > 512:
		common.Fail(c, common.CodeParamError, "OG 图不能超过 512 字符")
	default:
		return true
	}
	return false
}

func pageSlugTaken(slug string, excludeID uint) bool {
	var n int64
	model.DB.Model(&model.Page{}).Where("slug = ? AND id <> ?", slug, excludeID).Count(&n)
	return n > 0
}

// pageSlugTakenTx 事务内版本的 slug 占用检查
func pageSlugTakenTx(tx *gorm.DB, slug string, excludeID uint) bool {
	var n int64
	tx.Model(&model.Page{}).Where("slug = ? AND id <> ?", slug, excludeID).Count(&n)
	return n > 0
}

// derivePageSlug 由标题派生 slug（保留字母数字与 - _，其余转 -）；无法派生时用 page-{id}
func derivePageSlug(title string, id uint) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			prevDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return fmt.Sprintf("page-%d", id)
	}
	if len([]rune(s)) > 200 {
		s = string([]rune(s)[:200])
	}
	return s
}

// ensurePageSlug 保证 slug 唯一（冲突追加 -{id}，再冲突追加随机后缀）。
// 仅解析出最终 slug 并写回结构体，不落库——由调用方在事务内统一持久化。
func ensurePageSlug(tx *gorm.DB, page *model.Page, desired string) error {
	desired = strings.TrimSpace(desired)
	if desired == "" {
		desired = derivePageSlug(page.Title, page.ID)
	}
	if !pageSlugTakenTx(tx, desired, page.ID) {
		page.Slug = desired
		return nil
	}
	withID := fmt.Sprintf("%s-%d", desired, page.ID)
	if len([]rune(withID)) > 200 {
		withID = fmt.Sprintf("%s-%d", string([]rune(desired)[:190]), page.ID)
	}
	if !pageSlugTakenTx(tx, withID, page.ID) {
		page.Slug = withID
		return nil
	}
	page.Slug = fmt.Sprintf("%s-%s", withID, randomHex(4))
	return nil
}

// persistPageSlugIfChanged slug 与库中不一致时落库（创建/更新后调用）
func persistPageSlugIfChanged(tx *gorm.DB, page *model.Page, previous string) error {
	if page.Slug == previous {
		return nil
	}
	return tx.Model(page).UpdateColumn("slug", page.Slug).Error
}

// publishAtForPageStatus 非定时发布状态一律清空 publish_at。
//
// 写入前统一归一为 UTC：glebarez/sqlite 把 time.Time 落库为带时区的 RFC3339 字符串，
// 本地时间会写成 "...+08:00"、UTC 会写成 "...Z"；而 SQLite 的 `publish_at <= ?`
// 是**字典序字符串比较**，两种格式混用会产生错误结果（例如 UTC 的 03:09Z 会被判为
// 早于本地的 11:09+08:00，尽管二者是同一时刻）。
// 归一为 UTC 后，库内所有 publish_at 与调度器的比较基准格式一致，字典序等价于时间先后。
func publishAtForPageStatus(status int8, publishAt *time.Time) any {
	if status == model.PostScheduled && publishAt != nil {
		return publishAt.UTC()
	}
	return nil
}

// publishNowIfFirstPage 首次转为已发布时写入 published_at
func publishNowIfFirstPage(page *model.Page, status int8, updates map[string]any) {
	if status == model.PostPublished && page.PublishedAt == nil {
		now := time.Now()
		updates["published_at"] = now
	}
}

// maybeCreatePageRevisionOnUpdate 内容变化时生成版本（auto 保存含防抖）
func maybeCreatePageRevisionOnUpdate(tx *gorm.DB, page *model.Page, auto bool) error {
	latest, err := model.LatestPageRevision(tx, page.ID)
	if err != nil {
		return err
	}
	remark := model.PageRevisionRemark(page, latest)
	if remark == "" {
		return nil // 无变化
	}
	if auto && latest != nil && time.Since(latest.CreatedAt) < model.RevisionAutoThrottle {
		return nil // 自动保存防抖：仅保存内容，不生成版本
	}
	return model.CreatePageRevision(tx, page, remark)
}

// respondAdminPage 重新加载并返回页面详情
func respondAdminPage(c *gin.Context, id uint) {
	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}
	common.OK(c, gin.H{"page": model.ToPageDTO(&page)})
}

// AdminCreatePage POST /api/v1/admin/pages
func AdminCreatePage(c *gin.Context) {
	var req pagePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if !req.validate(c) {
		return
	}
	if req.Status == model.PostScheduled && !req.PublishAt.After(time.Now()) {
		common.Fail(c, common.CodeParamError, "publishAt 必须晚于当前时间")
		return
	}
	slug := strings.TrimSpace(req.Slug)
	if slug != "" && pageSlugTaken(slug, 0) {
		common.Fail(c, common.CodeParamError, "slug 已存在")
		return
	}

	page := model.Page{
		Title:          strings.TrimSpace(req.Title),
		Slug:           slug,
		Content:        req.Content,
		Status:         req.Status,
		PageType:       normalizePageType(req.PageType),
		PublishAt:      pagePtrForStatus(req.Status, req.PublishAt),
		SeoTitle:       req.SeoTitle,
		SeoDescription: req.SeoDescription,
		Canonical:      req.Canonical,
		OgImage:        req.OgImage,
	}
	if page.Status == model.PostPublished {
		now := time.Now()
		page.PublishedAt = &now
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&page).Error; err != nil {
			return err
		}
		if err := ensurePageSlug(tx, &page, slug); err != nil {
			return err
		}
		if err := persistPageSlugIfChanged(tx, &page, slug); err != nil {
			return err
		}
		return model.CreatePageRevision(tx, &page, "首次保存")
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	writeAudit(c, "page.create", "page", fmt.Sprint(page.ID), "创建页面："+page.Title)
	respondAdminPage(c, page.ID)
}

// AdminGetPage GET /api/v1/admin/pages/:id
func AdminGetPage(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}
	common.OK(c, gin.H{"page": model.ToPageDTO(&page)})
}

// AdminUpdatePage PUT /api/v1/admin/pages/:id
func AdminUpdatePage(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req pagePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if !req.validate(c) {
		return
	}

	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}

	// 并发编辑保护：基线不一致 → 10005，内容不被覆盖
	if req.BaseUpdatedAt != nil &&
		normalizeMillis(page.UpdatedAt).Sub(normalizeMillis(*req.BaseUpdatedAt)).Abs() > updateConflictTolerance {
		common.FailWithData(c, common.CodeConflict, "页面已在其他窗口被修改，请刷新后重试", gin.H{
			"currentUpdatedAt": page.UpdatedAt,
			"currentTitle":     page.Title,
		})
		return
	}

	if req.Status == model.PostScheduled && !req.PublishAt.After(time.Now()) {
		common.Fail(c, common.CodeParamError, "publishAt 必须晚于当前时间")
		return
	}

	slug := strings.TrimSpace(req.Slug)
	if slug != "" && pageSlugTaken(slug, page.ID) {
		common.Fail(c, common.CodeParamError, "slug 已存在")
		return
	}
	oldSlug := page.Slug
	// 事务内 slug 变化的落库基线（与 oldSlug 相同语义，单独命名以区分「无 slug 变化」场景）
	oldSlugSnapshot := page.Slug

	now := normalizeMillis(time.Now())
	updates := map[string]any{
		"title":           strings.TrimSpace(req.Title),
		"content":         req.Content,
		"status":          req.Status,
		"page_type":       normalizePageType(req.PageType),
		"publish_at":      publishAtForPageStatus(req.Status, req.PublishAt),
		"seo_title":       req.SeoTitle,
		"seo_description": req.SeoDescription,
		"canonical":       req.Canonical,
		"og_image":        req.OgImage,
		"updated_at":      now,
	}
	publishNowIfFirstPage(&page, req.Status, updates)

	var newSlug string
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		// 显式 Omit(updated_at) 后统一在末尾回写，规避 GORM 钩子二次覆盖
		if err := tx.Model(&page).Omit("updated_at").Updates(updates).Error; err != nil {
			return err
		}
		// slug：同步结构体后保证唯一（冲突追加后缀）
		page.Title = strings.TrimSpace(req.Title)
		page.Content = req.Content
		page.Status = req.Status
		page.PageType = normalizePageType(req.PageType)
		page.SeoTitle = req.SeoTitle
		page.SeoDescription = req.SeoDescription
		page.Canonical = req.Canonical
		page.OgImage = req.OgImage
		page.PublishAt = pagePtrForStatus(req.Status, req.PublishAt)
		if err := ensurePageSlug(tx, &page, slug); err != nil {
			return err
		}
		if err := persistPageSlugIfChanged(tx, &page, oldSlugSnapshot); err != nil {
			return err
		}
		newSlug = page.Slug

		// slug 变更 → 自动 301（复用既有 redirects 系统与开关）
		if oldSlug != "" && newSlug != oldSlug && autoRedirectOnSlugChangeEnabled(tx) {
			source, target := "/page/"+oldSlug, "/page/"+newSlug
			if !model.RedirectLoopExists(tx, source, target) {
				if err := model.UpsertRedirectForSlug(tx, source, target); err != nil {
					return err
				}
			}
		}

		if err := tx.Model(&page).UpdateColumn("updated_at", now).Error; err != nil {
			return err
		}
		// 时间戳同步，供版本比较与响应使用
		page.UpdatedAt = now
		if pub, ok := updates["published_at"].(time.Time); ok {
			p := pub
			page.PublishedAt = &p
		}
		return maybeCreatePageRevisionOnUpdate(tx, &page, req.Auto)
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}

	writeAudit(c, "page.update", "page", fmt.Sprint(page.ID), "更新页面："+page.Title)
	if oldSlug != "" && newSlug != oldSlug {
		writeAudit(c, "page.slug_change", "page", fmt.Sprint(page.ID), fmt.Sprintf("slug 变更：%s → %s", oldSlug, newSlug))
	}
	respondAdminPage(c, page.ID)
}

// AdminUpdatePageStatus PUT /api/v1/admin/pages/:id/status（快速编辑）
func AdminUpdatePageStatus(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req struct {
		Status    int8       `json:"status"`
		PublishAt *time.Time `json:"publishAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if req.Status < model.PostDraft || req.Status > model.PostScheduled {
		common.Fail(c, common.CodeParamError, "status 仅允许 0 / 1 / 2 / 3")
		return
	}
	if req.Status == model.PostScheduled && req.PublishAt == nil {
		common.Fail(c, common.CodeParamError, "定时发布必须提供 publishAt")
		return
	}

	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}

	now := normalizeMillis(time.Now())
	updates := map[string]any{
		"status":     req.Status,
		"publish_at": publishAtForPageStatus(req.Status, req.PublishAt),
		"updated_at": now,
	}
	publishNowIfFirstPage(&page, req.Status, updates)

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&page).Omit("updated_at").Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Model(&page).UpdateColumn("updated_at", now).Error; err != nil {
			return err
		}
		page.UpdatedAt = now
		page.Status = req.Status
		page.PublishAt = pagePtrForStatus(req.Status, req.PublishAt)
		if pub, ok := updates["published_at"].(time.Time); ok {
			p := pub
			page.PublishedAt = &p
		}
		return maybeCreatePageRevisionOnUpdate(tx, &page, false)
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	writeAudit(c, "page.status", "page", fmt.Sprint(page.ID), fmt.Sprintf("页面状态改为 %d：%s", req.Status, page.Title))
	respondAdminPage(c, page.ID)
}

// AdminCopyPage POST /api/v1/admin/pages/:id/copy
func AdminCopyPage(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var src model.Page
	if err := model.DB.First(&src, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}

	page := model.Page{
		Title:          src.Title + "（副本）",
		Content:        src.Content,
		Status:         model.PostDraft, // 副本绝不能直接公开
		PageType:       src.PageType,
		SeoTitle:       src.SeoTitle,
		SeoDescription: src.SeoDescription,
		Canonical:      src.Canonical,
		OgImage:        src.OgImage,
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&page).Error; err != nil {
			return err
		}
		if err := ensurePageSlug(tx, &page, src.Slug+"-copy"); err != nil {
			return err
		}
		if err := tx.Model(&page).UpdateColumn("slug", page.Slug).Error; err != nil {
			return err
		}
		return model.CreatePageRevision(tx, &page, "首次保存")
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	writeAudit(c, "page.copy", "page", fmt.Sprint(page.ID), "复制页面："+src.Title+" → "+page.Title)
	respondAdminPage(c, page.ID)
}

// AdminBatchPages POST /api/v1/admin/pages/batch
func AdminBatchPages(c *gin.Context) {
	var req struct {
		Action string `json:"action"`
		IDs    []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if len(req.IDs) == 0 {
		common.Fail(c, common.CodeParamError, "ids 不能为空")
		return
	}
	if len(req.IDs) > 100 {
		common.Fail(c, common.CodeParamError, "ids 不能超过 100 个")
		return
	}
	switch req.Action {
	case "publish", "hide", "delete":
	default:
		common.Fail(c, common.CodeParamError, "action 仅允许 publish / hide / delete")
		return
	}

	var updated int64
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		switch req.Action {
		case "publish":
			now := normalizeMillis(time.Now())
			res := tx.Model(&model.Page{}).Where("id IN ?", req.IDs).
				Updates(map[string]any{"status": model.PostPublished, "publish_at": nil, "updated_at": now})
			if res.Error != nil {
				return res.Error
			}
			// 首次发布的行补 published_at（null 才写，保留既有发布时间）
			if err := tx.Model(&model.Page{}).Where("id IN ? AND published_at IS NULL", req.IDs).
				Update("published_at", now).Error; err != nil {
				return err
			}
			updated = res.RowsAffected
		case "hide":
			now := normalizeMillis(time.Now())
			res := tx.Model(&model.Page{}).Where("id IN ?", req.IDs).
				Updates(map[string]any{"status": model.PostHidden, "publish_at": nil, "updated_at": now})
			if res.Error != nil {
				return res.Error
			}
			updated = res.RowsAffected
		case "delete":
			res := tx.Where("id IN ?", req.IDs).Delete(&model.Page{})
			if res.Error != nil {
				return res.Error
			}
			if err := tx.Where("page_id IN ?", req.IDs).Delete(&model.PageRevision{}).Error; err != nil {
				return err
			}
			updated = res.RowsAffected
		}
		return nil
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	writeAudit(c, "page.batch", "page", "", fmt.Sprintf("批量%s %d 个页面", req.Action, updated))
	common.OK(c, gin.H{"updated": updated})
}

// AdminDeletePage DELETE /api/v1/admin/pages/:id
func AdminDeletePage(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}
	title := page.Title
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("page_id = ?", page.ID).Delete(&model.PageRevision{}).Error; err != nil {
			return err
		}
		return tx.Delete(&page).Error
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	writeAudit(c, "page.delete", "page", fmt.Sprint(id), "删除页面："+title)
	common.OK(c, nil)
}

// AdminPreviewPage GET /api/v1/admin/pages/preview/:id
// 未发布页面的后台预览数据源（公开 #8 只认 status=1）
func AdminPreviewPage(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var page model.Page
	if err := model.DB.First(&page, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}
	common.OK(c, gin.H{"page": model.ToPageDTO(&page)})
}

// pagePtrForStatus 仅定时发布携带 publishAt；落库前统一归一为 UTC。
// 归一原因见 publishAtForPageStatus —— 调度器以 UTC 做字典序比较，
// 写入侧若保留本地时区格式会与之失配，导致未到点就被发布。
func pagePtrForStatus(status int8, publishAt *time.Time) *time.Time {
	if status == model.PostScheduled && publishAt != nil {
		utc := publishAt.UTC()
		return &utc
	}
	return nil
}
