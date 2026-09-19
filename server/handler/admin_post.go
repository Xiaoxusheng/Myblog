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

// postPayload 创建/更新文章入参（契约 #17/#19）
type postPayload struct {
	Title      string     `json:"title"`
	Slug       string     `json:"slug"`
	Summary    string     `json:"summary"`
	Content    string     `json:"content"`
	Cover      string     `json:"cover"`
	CategoryID uint       `json:"categoryId"`
	Tags       []string   `json:"tags"`
	Status     int8       `json:"status"`
	IsTop      bool       `json:"isTop"`
	PublishAt  *time.Time `json:"publishAt"`  // 定时发布计划时间；status=3 必填
	Auto       bool       `json:"auto"`       // 前端自动保存标记：版本生成防抖
	SeriesID   uint       `json:"seriesId"`   // 0=移出专题
	SeriesSort int        `json:"seriesSort"` // 0=自动排到末尾（已是成员则保持原序号）
}

func (p *postPayload) validate(c *gin.Context) bool {
	switch {
	case strings.TrimSpace(p.Title) == "":
		common.Fail(c, common.CodeParamError, "标题不能为空")
	case utf8.RuneCountInString(p.Title) > 200:
		common.Fail(c, common.CodeParamError, "标题不能超过 200 字")
	case utf8.RuneCountInString(p.Summary) > 1000:
		common.Fail(c, common.CodeParamError, "摘要不能超过 1000 字")
	case utf8.RuneCountInString(strings.TrimSpace(p.Slug)) > 200:
		common.Fail(c, common.CodeParamError, "slug 不能超过 200 字符")
	case utf8.RuneCountInString(p.Cover) > 512:
		common.Fail(c, common.CodeParamError, "封面地址过长")
	case p.Status < model.PostDraft || p.Status > model.PostScheduled:
		common.Fail(c, common.CodeParamError, "status 取值不合法")
	case p.Status == model.PostScheduled && p.PublishAt == nil:
		common.Fail(c, common.CodeParamError, "定时发布需要计划发布时间")
	default:
		return true
	}
	return false
}

// AdminListPosts GET /api/v1/admin/posts —— keyword/status/categoryId 过滤，最新在前
func AdminListPosts(c *gin.Context) {
	pq := common.ParsePage(c, 10)

	statusParam := c.Query("status")
	if statusParam != "" {
		if s, err := parsePostStatus(statusParam); err != nil || s < 0 {
			common.Fail(c, common.CodeParamError, "status 参数不合法")
			return
		}
	}

	buildQuery := func() *gorm.DB {
		db := model.DB.Model(&model.Post{})
		if kw := strings.TrimSpace(c.Query("keyword")); kw != "" {
			like := likeContains(kw)
			db = db.Where(
				"(title "+likeEscapeClause+" OR summary "+likeEscapeClause+" OR content "+likeEscapeClause+")",
				like, like, like,
			)
		}
		if statusParam != "" {
			s, _ := parsePostStatus(statusParam)
			db = db.Where("status = ?", s)
		}
		if catID := queryUint(c, "categoryId"); catID > 0 {
			db = db.Where("category_id = ?", catID)
		}
		return db.Order("created_at DESC, id DESC")
	}

	var total int64
	if err := buildQuery().Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var posts []model.Post
	if err := buildQuery().Preload("Category").Preload("Tags").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&posts).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	ids := make([]uint, 0, len(posts))
	for i := range posts {
		ids = append(ids, posts[i].ID)
	}
	counts := commentCountsByPost(ids)

	list := make([]model.AdminPostItemDTO, 0, len(posts))
	for i := range posts {
		list = append(list, model.ToAdminPostItem(&posts[i], counts[posts[i].ID]))
	}
	common.OK(c, pq.Data(list, total))
}

func parsePostStatus(raw string) (int8, error) {
	switch raw {
	case "0":
		return model.PostDraft, nil
	case "1":
		return model.PostPublished, nil
	case "2":
		return model.PostHidden, nil
	case "3":
		return model.PostScheduled, nil
	default:
		return -1, fmt.Errorf("invalid status %q", raw)
	}
}

func commentCountsByPost(postIDs []uint) map[uint]int64 {
	type row struct {
		PostID uint  `gorm:"column:post_id"`
		Cnt    int64 `gorm:"column:cnt"`
	}
	counts := make(map[uint]int64, len(postIDs))
	if len(postIDs) == 0 {
		return counts
	}
	var rows []row
	if err := model.DB.Model(&model.Comment{}).
		Select("post_id, COUNT(*) AS cnt").
		Where("post_id IN ?", postIDs).
		Group("post_id").Scan(&rows).Error; err != nil {
		return counts
	}
	for _, r := range rows {
		counts[r.PostID] = r.Cnt
	}
	return counts
}

// AdminGetPost GET /api/v1/admin/posts/:id
func AdminGetPost(c *gin.Context) {
	post, ok := fetchPostWithAssoc(c.Param("id"))
	if !ok {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}
	counts := commentCountsByPost([]uint{post.ID})
	common.OK(c, gin.H{"post": model.ToAdminPostItem(post, counts[post.ID])})
}

func fetchPostWithAssoc(idParam string) (*model.Post, bool) {
	id, err := strconvUint(idParam)
	if err != nil {
		return nil, false
	}
	var post model.Post
	if err := model.DB.Preload("Category").Preload("Tags").First(&post, id).Error; err != nil {
		return nil, false
	}
	return &post, true
}

func strconvUint(s string) (uint, error) {
	var v uint64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a number: %q", s)
		}
		v = v*10 + uint64(r-'0')
		if v > 1<<62 {
			return 0, fmt.Errorf("number too large: %q", s)
		}
	}
	if s == "" {
		return 0, fmt.Errorf("empty number")
	}
	return uint(v), nil
}

// AdminCreatePost POST /api/v1/admin/posts
func AdminCreatePost(c *gin.Context) {
	var req postPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if !req.validate(c) {
		return
	}
	if req.CategoryID > 0 {
		var n int64
		model.DB.Model(&model.Category{}).Where("id = ?", req.CategoryID).Count(&n)
		if n == 0 {
			common.Fail(c, common.CodeParamError, "分类不存在")
			return
		}
	}
	if err := checkSeriesExists(req.SeriesID); err != nil {
		common.Fail(c, common.CodeParamError, "专题不存在")
		return
	}

	post := model.Post{
		Title:      strings.TrimSpace(req.Title),
		Summary:    strings.TrimSpace(req.Summary),
		Content:    req.Content,
		Cover:      strings.TrimSpace(req.Cover),
		CategoryID: req.CategoryID,
		Status:     req.Status,
		IsTop:      req.IsTop,
		SeriesID:   req.SeriesID,
		// 先用一次性临时占位 slug 插入（uniqueIndex 冲突规避），随后按契约规则定稿
		Slug: fmt.Sprintf("post-tmp-%d-%s", time.Now().UnixNano(), randomHex(4)),
	}
	if req.Status == model.PostPublished {
		post.PublishedAt = nowPtr()
	}
	if req.Status == model.PostScheduled {
		post.PublishAt = req.PublishAt
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		if err := ensurePostSlug(tx, &post, req.Slug); err != nil {
			return err
		}
		tags, err := upsertTagsByName(tx, req.Tags)
		if err != nil {
			return err
		}
		if err := tx.Model(&post).Association("Tags").Replace(tags); err != nil {
			return err
		}
		if req.SeriesID > 0 {
			sort, err := nextSeriesSort(tx, req.SeriesID, req.SeriesSort)
			if err != nil {
				return err
			}
			post.SeriesSort = sort
			if err := tx.Model(&post).Update("series_sort", sort).Error; err != nil {
				return err
			}
		}
		// slug 已定稿后写入首个版本（契约：创建即 v1，remark=首次保存）
		return model.CreatePostRevision(tx, &post, "首次保存")
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}

	// 重新加载关联后返回
	if err := model.DB.Preload("Category").Preload("Tags").First(&post, post.ID).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	counts := commentCountsByPost([]uint{post.ID})
	common.OK(c, gin.H{"post": model.ToAdminPostItem(&post, counts[post.ID])})
}

// AdminUpdatePost PUT /api/v1/admin/posts/:id —— 全量更新
func AdminUpdatePost(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "文章 id 不合法")
		return
	}
	var req postPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if !req.validate(c) {
		return
	}
	if req.CategoryID > 0 {
		var n int64
		model.DB.Model(&model.Category{}).Where("id = ?", req.CategoryID).Count(&n)
		if n == 0 {
			common.Fail(c, common.CodeParamError, "分类不存在")
			return
		}
	}

	var post model.Post
	if err := model.DB.First(&post, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}
	oldSlug := post.Slug
	if err := checkSeriesExists(req.SeriesID); err != nil {
		common.Fail(c, common.CodeParamError, "专题不存在")
		return
	}

	updates := map[string]any{
		"title":       strings.TrimSpace(req.Title),
		"summary":     strings.TrimSpace(req.Summary),
		"content":     req.Content,
		"cover":       strings.TrimSpace(req.Cover),
		"category_id": req.CategoryID,
		"status":      req.Status,
		"is_top":      req.IsTop,
		// 定时发布写计划时间；离开定时状态置空
		"publish_at": publishAtForStatus(req.Status, req.PublishAt),
	}
	if err := applySeriesAssignment(model.DB, &post, &req, updates); err != nil {
		common.ServerError(c, err)
		return
	}
	publishNowIfFirst(&post, req.Status, updates)

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&post).Updates(updates).Error; err != nil {
			return err
		}
		// Updates 不回写结构体，手动同步 published_at 供后续判断
		if pa, ok := updates["published_at"]; ok {
			t := pa.(time.Time)
			post.PublishedAt = &t
		}
		if err := ensurePostSlug(tx, &post, req.Slug); err != nil {
			return err
		}
		// slug 变更且开关开启 → 自动创建旧→新 301 重定向（契约 #19）
		if oldSlug != "" && post.Slug != oldSlug && autoRedirectOnSlugChangeEnabled(tx) {
			src, tgt := "/post/"+oldSlug, "/post/"+post.Slug
			if !model.RedirectLoopExists(tx, src, tgt) {
				if err := model.UpsertRedirectForSlug(tx, src, tgt); err != nil {
					return err
				}
			}
		}
		// 同步结构体中的专题字段供响应与后续逻辑
		if v, ok := updates["series_id"].(uint); ok {
			post.SeriesID = v
		}
		if v, ok := updates["series_sort"].(int); ok {
			post.SeriesSort = v
		}
		tags, err := upsertTagsByName(tx, req.Tags)
		if err != nil {
			return err
		}
		if err := tx.Model(&post).Association("Tags").Replace(tags); err != nil {
			return err
		}
		return maybeCreateRevisionOnUpdate(tx, &post, &req)
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}

	if err := model.DB.Preload("Category").Preload("Tags").First(&post, post.ID).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	counts := commentCountsByPost([]uint{post.ID})
	common.OK(c, gin.H{"post": model.ToAdminPostItem(&post, counts[post.ID])})
}

// AdminUpdatePostStatus PUT /api/v1/admin/posts/:id/status
func AdminUpdatePostStatus(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "文章 id 不合法")
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
		common.Fail(c, common.CodeParamError, "status 取值不合法")
		return
	}
	if req.Status == model.PostScheduled && req.PublishAt == nil {
		common.Fail(c, common.CodeParamError, "定时发布需要计划发布时间")
		return
	}

	var post model.Post
	if err := model.DB.First(&post, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}

	updates := map[string]any{
		"status":     req.Status,
		"publish_at": publishAtForStatus(req.Status, req.PublishAt),
	}
	publishNowIfFirst(&post, req.Status, updates)

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&post).Updates(updates).Error; err != nil {
			return err
		}
		// 状态未变化（仅调整计划时间）不生成版本
		if post.Status == req.Status {
			return nil
		}
		latest, err := model.LatestPostRevision(tx, post.ID)
		if err != nil {
			return err
		}
		newState := post
		newState.Status = req.Status
		remark := model.RevisionRemark(&newState, latest)
		if remark == "" {
			return nil
		}
		return model.CreatePostRevision(tx, &newState, remark)
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// AdminDeletePost DELETE /api/v1/admin/posts/:id —— 连带 post_tags、评论与版本历史
func AdminDeletePost(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "文章 id 不合法")
		return
	}
	var post model.Post
	if err := model.DB.First(&post, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "文章不存在")
		return
	}

	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&post).Association("Tags").Clear(); err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", post.ID).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", post.ID).Delete(&model.PostRevision{}).Error; err != nil {
			return err
		}
		return tx.Delete(&post).Error
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// publishAtForStatus publish_at 写库语义：status=3 写计划时间，其余状态置空（NULL）
func publishAtForStatus(status int8, publishAt *time.Time) *time.Time {
	if status == model.PostScheduled {
		return publishAt
	}
	return nil
}

// maybeCreateRevisionOnUpdate 保存后按契约生成文章版本：
// 快照字段与最新版本相同不生成；auto 保存且距最新版本不足防抖间隔（RevisionAutoThrottle）不生成。
func maybeCreateRevisionOnUpdate(tx *gorm.DB, post *model.Post, req *postPayload) error {
	latest, err := model.LatestPostRevision(tx, post.ID)
	if err != nil {
		return err
	}
	// Updates(map) 不回写结构体，新状态以当前文章为基础覆盖请求字段
	// （保留 ID；Slug 已由 ensurePostSlug 定稿回写 post）
	newState := *post
	newState.Title = strings.TrimSpace(req.Title)
	newState.Summary = strings.TrimSpace(req.Summary)
	newState.Content = req.Content
	newState.Cover = strings.TrimSpace(req.Cover)
	newState.CategoryID = req.CategoryID
	newState.IsTop = req.IsTop
	newState.Status = req.Status
	remark := model.RevisionRemark(&newState, latest)
	if remark == "" {
		return nil
	}
	if req.Auto && latest != nil && time.Since(latest.CreatedAt) < model.RevisionAutoThrottle {
		return nil
	}
	return model.CreatePostRevision(tx, &newState, remark)
}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}
