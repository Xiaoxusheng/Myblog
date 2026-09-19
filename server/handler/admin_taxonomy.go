package handler

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------- 分类（契约 #22-25） ----------

// AdminListCategories GET /api/v1/admin/categories —— 分页，含 postCount（全部文章口径）
func AdminListCategories(c *gin.Context) {
	pq := common.ParsePage(c, 10)

	var total int64
	if err := model.DB.Model(&model.Category{}).Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var categories []model.Category
	if err := model.DB.Order("id ASC").Offset(pq.Offset()).Limit(pq.PageSize).
		Find(&categories).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	fillCategoryPostCounts(categories, false)
	common.OK(c, pq.Data(categories, total))
}

type taxonomyPayload struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// AdminCreateCategory POST /api/v1/admin/categories —— name 唯一
func AdminCreateCategory(c *gin.Context) {
	var req taxonomyPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		common.Fail(c, common.CodeParamError, "分类名称不能为空")
		return
	}
	if utf8.RuneCountInString(req.Name) > 64 || utf8.RuneCountInString(req.Description) > 500 {
		common.Fail(c, common.CodeParamError, "字段长度超出限制")
		return
	}
	var n int64
	model.DB.Model(&model.Category{}).Where("name = ?", req.Name).Count(&n)
	if n > 0 {
		common.Fail(c, common.CodeParamError, "分类名称已存在")
		return
	}

	category := model.Category{
		Name:        req.Name,
		Slug:        taxonomySlug(req.Slug, req.Name),
		Description: strings.TrimSpace(req.Description),
	}
	if err := model.DB.Create(&category).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	if category.Slug == "" {
		category.Slug = "cat-" + strconv.Itoa(int(category.ID))
		if err := model.DB.Model(&category).Update("slug", category.Slug).Error; err != nil {
			common.ServerError(c, err)
			return
		}
	}
	common.OK(c, category)
}

// AdminUpdateCategory PUT /api/v1/admin/categories/:id
func AdminUpdateCategory(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req taxonomyPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		common.Fail(c, common.CodeParamError, "分类名称不能为空")
		return
	}
	if utf8.RuneCountInString(req.Name) > 64 || utf8.RuneCountInString(req.Description) > 500 {
		common.Fail(c, common.CodeParamError, "字段长度超出限制")
		return
	}

	var category model.Category
	if err := model.DB.First(&category, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "分类不存在")
		return
	}
	var n int64
	model.DB.Model(&model.Category{}).Where("name = ? AND id <> ?", req.Name, category.ID).Count(&n)
	if n > 0 {
		common.Fail(c, common.CodeParamError, "分类名称已存在")
		return
	}

	updates := map[string]any{
		"name":        req.Name,
		"description": strings.TrimSpace(req.Description),
	}
	if slug := taxonomySlug(req.Slug, req.Name); slug != "" {
		updates["slug"] = slug
	}
	if err := model.DB.Model(&category).Updates(updates).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, category)
}

// AdminDeleteCategory DELETE /api/v1/admin/categories/:id —— 其下文章 categoryId 置 0
func AdminDeleteCategory(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var category model.Category
	if err := model.DB.First(&category, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "分类不存在")
		return
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Post{}).Where("category_id = ?", category.ID).
			Update("category_id", 0).Error; err != nil {
			return err
		}
		return tx.Delete(&category).Error
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// ---------- 标签（契约 #26-29） ----------

// AdminListTags GET /api/v1/admin/tags —— 分页（契约默认 100，受全局上限 50 约束），含 postCount
func AdminListTags(c *gin.Context) {
	pq := common.ParsePage(c, 100)

	var total int64
	if err := model.DB.Model(&model.Tag{}).Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var tags []model.Tag
	if err := model.DB.Order("id ASC").Offset(pq.Offset()).Limit(pq.PageSize).
		Find(&tags).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	fillTagPostCounts(tags, false)
	common.OK(c, pq.Data(tags, total))
}

type tagPayload struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// AdminCreateTag POST /api/v1/admin/tags
func AdminCreateTag(c *gin.Context) {
	var req tagPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		common.Fail(c, common.CodeParamError, "标签名称不能为空")
		return
	}
	if utf8.RuneCountInString(req.Name) > 64 {
		common.Fail(c, common.CodeParamError, "标签名称过长")
		return
	}
	var n int64
	model.DB.Model(&model.Tag{}).Where("name = ?", req.Name).Count(&n)
	if n > 0 {
		common.Fail(c, common.CodeParamError, "标签名称已存在")
		return
	}

	tag := model.Tag{Name: req.Name, Slug: taxonomySlug(req.Slug, req.Name)}
	if err := model.DB.Create(&tag).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	if tag.Slug == "" {
		tag.Slug = "tag-" + strconv.Itoa(int(tag.ID))
		if err := model.DB.Model(&tag).Update("slug", tag.Slug).Error; err != nil {
			common.ServerError(c, err)
			return
		}
	}
	common.OK(c, tag)
}

// AdminUpdateTag PUT /api/v1/admin/tags/:id
func AdminUpdateTag(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req tagPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		common.Fail(c, common.CodeParamError, "标签名称不能为空")
		return
	}
	if utf8.RuneCountInString(req.Name) > 64 {
		common.Fail(c, common.CodeParamError, "标签名称过长")
		return
	}

	var tag model.Tag
	if err := model.DB.First(&tag, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "标签不存在")
		return
	}
	var n int64
	model.DB.Model(&model.Tag{}).Where("name = ? AND id <> ?", req.Name, tag.ID).Count(&n)
	if n > 0 {
		common.Fail(c, common.CodeParamError, "标签名称已存在")
		return
	}

	updates := map[string]any{"name": req.Name}
	if slug := taxonomySlug(req.Slug, req.Name); slug != "" {
		updates["slug"] = slug
	}
	if err := model.DB.Model(&tag).Updates(updates).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, tag)
}

// AdminDeleteTag DELETE /api/v1/admin/tags/:id —— 同步清理 post_tags
func AdminDeleteTag(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var tag model.Tag
	if err := model.DB.First(&tag, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "标签不存在")
		return
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tag_id = ?", tag.ID).Delete(&model.PostTag{}).Error; err != nil {
			return err
		}
		return tx.Delete(&tag).Error
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// ---------- 通用助手 ----------

// parseIDParam 解析 :id 路径参数；非法时直接回复 10001
func parseIDParam(c *gin.Context) (uint, bool) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "id 不合法")
		return 0, false
	}
	return id, true
}
