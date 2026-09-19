package handler

// 管理端专题管理（契约 #56-61）+ 文章-专题归属辅助函数（admin_post 使用）

import (
	"fmt"
	"strings"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------- 文章-专题归属辅助 ----------

// checkSeriesExists seriesId>0 时校验专题存在；0 视为移出专题
func checkSeriesExists(seriesID uint) error {
	if seriesID == 0 {
		return nil
	}
	var n int64
	model.DB.Model(&model.Series{}).Where("id = ?", seriesID).Count(&n)
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// nextSeriesSort 计算文章在专题内的序号：显式序号>0 用显式值，否则排到末尾
func nextSeriesSort(db *gorm.DB, seriesID uint, desired int) (int, error) {
	if desired > 0 {
		return desired, nil
	}
	var maxSort int
	if err := db.Model(&model.Post{}).Where("series_id = ?", seriesID).
		Select("COALESCE(MAX(series_sort), 0)").Scan(&maxSort).Error; err != nil {
		return 0, err
	}
	return maxSort + 1, nil
}

// applySeriesAssignment 计算更新时的专题归属写入值（契约 #19）：
// seriesId 变更 → 新专题（sort 0=自动排末尾）；同专题且显式给序号 → 调整序号；0 → 移出。
func applySeriesAssignment(db *gorm.DB, post *model.Post, req *postPayload, updates map[string]any) error {
	switch {
	case req.SeriesID == 0:
		updates["series_id"] = uint(0)
		updates["series_sort"] = 0
	case post.SeriesID != req.SeriesID:
		sort, err := nextSeriesSort(db, req.SeriesID, req.SeriesSort)
		if err != nil {
			return err
		}
		updates["series_id"] = req.SeriesID
		updates["series_sort"] = sort
	default:
		sort := post.SeriesSort
		if req.SeriesSort > 0 {
			sort = req.SeriesSort
		}
		updates["series_id"] = req.SeriesID
		updates["series_sort"] = sort
	}
	return nil
}

// ---------- 管理端专题接口 ----------

// seriesCountMap 批量统计专题文章数（admin 传全部状态，public 传已发布）
func seriesCountMap(statusFilter *int8) map[uint]int64 {
	type row struct {
		SeriesID uint  `gorm:"column:series_id"`
		Cnt      int64 `gorm:"column:cnt"`
	}
	query := model.DB.Model(&model.Post{}).Where("series_id > 0")
	if statusFilter != nil {
		query = query.Where("status = ?", *statusFilter)
	}
	var rows []row
	if err := query.Select("series_id, COUNT(*) AS cnt").Group("series_id").Scan(&rows).Error; err != nil {
		return map[uint]int64{}
	}
	counts := make(map[uint]int64, len(rows))
	for _, r := range rows {
		counts[r.SeriesID] = r.Cnt
	}
	return counts
}

// seriesDTO 契约 Series 对象
type seriesDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Cover       string `json:"cover"`
	Visible     bool   `json:"visible"`
	Sort        int    `json:"sort"`
	PostCount   int64  `json:"postCount"`
}

func toSeriesDTO(s *model.Series, count int64) seriesDTO {
	return seriesDTO{
		ID: s.ID, Name: s.Name, Slug: s.Slug, Description: s.Description,
		Cover: s.Cover, Visible: s.Visible, Sort: s.Sort, PostCount: count,
	}
}

// seriesPayload 创建/更新专题入参（契约 #57/#58）
type seriesPayload struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Cover       string `json:"cover"`
	Visible     bool   `json:"visible"`
	Sort        int    `json:"sort"`
}

func (p *seriesPayload) validate(c *gin.Context) bool {
	switch {
	case strings.TrimSpace(p.Name) == "":
		common.Fail(c, common.CodeParamError, "专题名称不能为空")
	case len([]rune(strings.TrimSpace(p.Name))) > 100:
		common.Fail(c, common.CodeParamError, "专题名称不能超过 100 字")
	case len([]rune(strings.TrimSpace(p.Slug))) > 100:
		common.Fail(c, common.CodeParamError, "slug 不能超过 100 字符")
	case len([]rune(p.Description)) > 500:
		common.Fail(c, common.CodeParamError, "描述不能超过 500 字")
	case len([]rune(p.Cover)) > 512:
		common.Fail(c, common.CodeParamError, "封面地址过长")
	default:
		return true
	}
	return false
}

// AdminListSeries GET /api/v1/admin/series —— sort 升序，postCount=全部文章数
func AdminListSeries(c *gin.Context) {
	pq := common.ParsePage(c, 50)

	var total int64
	if err := model.DB.Model(&model.Series{}).Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var list []model.Series
	if err := model.DB.Order("sort ASC, id ASC").
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&list).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	counts := seriesCountMap(nil)
	dtos := make([]seriesDTO, 0, len(list))
	for i := range list {
		dtos = append(dtos, toSeriesDTO(&list[i], counts[list[i].ID]))
	}
	common.OK(c, pq.Data(dtos, total))
}

// ensureSeriesSlug 计算并落库专题最终 slug（空→由名称派生/series-{id}；冲突→追加 -id）
func ensureSeriesSlug(db *gorm.DB, series *model.Series, desired string) error {
	final := strings.TrimSpace(desired)
	if final == "" {
		final = common.Slugify(series.Name)
	}
	taken := func(slug string) bool {
		var n int64
		db.Model(&model.Series{}).Where("slug = ? AND id <> ?", slug, series.ID).Count(&n)
		return n > 0
	}
	if final == "" {
		final = fmt.Sprintf("series-%d", series.ID)
	}
	if taken(final) {
		final = fmt.Sprintf("%s-%d", final, series.ID)
	}
	if taken(final) {
		final = fmt.Sprintf("series-%d-%s", series.ID, randomHex(4))
	}
	series.Slug = final
	return db.Model(series).Update("slug", final).Error
}

// AdminCreateSeries POST /api/v1/admin/series
func AdminCreateSeries(c *gin.Context) {
	var req seriesPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if !req.validate(c) {
		return
	}
	var n int64
	model.DB.Model(&model.Series{}).Where("name = ?", strings.TrimSpace(req.Name)).Count(&n)
	if n > 0 {
		common.Fail(c, common.CodeParamError, "专题名称已存在")
		return
	}
	series := model.Series{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Cover:       strings.TrimSpace(req.Cover),
		Visible:     req.Visible,
		Sort:        req.Sort,
		// 占位 slug，事务内定稿
		Slug: "series-tmp-" + randomHex(4),
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&series).Error; err != nil {
			return err
		}
		return ensureSeriesSlug(tx, &series, req.Slug)
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"series": toSeriesDTO(&series, 0)})
}

// AdminUpdateSeries PUT /api/v1/admin/series/:id
func AdminUpdateSeries(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "专题 id 不合法")
		return
	}
	var req seriesPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if !req.validate(c) {
		return
	}
	var series model.Series
	if err := model.DB.First(&series, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "专题不存在")
		return
	}
	var n int64
	model.DB.Model(&model.Series{}).Where("name = ? AND id <> ?", strings.TrimSpace(req.Name), series.ID).Count(&n)
	if n > 0 {
		common.Fail(c, common.CodeParamError, "专题名称已存在")
		return
	}

	updates := map[string]any{
		"name":        strings.TrimSpace(req.Name),
		"description": strings.TrimSpace(req.Description),
		"cover":       strings.TrimSpace(req.Cover),
		"visible":     req.Visible,
		"sort":        req.Sort,
	}
	if err := model.DB.Model(&series).Updates(updates).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	if err := ensureSeriesSlug(model.DB, &series, req.Slug); err != nil {
		common.ServerError(c, err)
		return
	}
	var count int64
	model.DB.Model(&model.Post{}).Where("series_id = ?", series.ID).Count(&count)
	common.OK(c, gin.H{"series": toSeriesDTO(&series, count)})
}

// AdminDeleteSeries DELETE /api/v1/admin/series/:id —— 成员文章 series_id 置 0
func AdminDeleteSeries(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "专题 id 不合法")
		return
	}
	var series model.Series
	if err := model.DB.First(&series, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "专题不存在")
		return
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Post{}).Where("series_id = ?", series.ID).
			Updates(map[string]any{"series_id": 0, "series_sort": 0}).Error; err != nil {
			return err
		}
		return tx.Delete(&series).Error
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// AdminSeriesPosts GET /api/v1/admin/series/:id/posts —— 全部状态，序号升序
func AdminSeriesPosts(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "专题 id 不合法")
		return
	}
	var series model.Series
	if err := model.DB.First(&series, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "专题不存在")
		return
	}
	type seriesPostItem struct {
		ID     uint   `json:"id"`
		Title  string `json:"title"`
		Slug   string `json:"slug"`
		Status int8   `json:"status"`
		Sort   int    `json:"sort"`
	}
	var posts []seriesPostItem
	if err := model.DB.Model(&model.Post{}).
		Select("id, title, slug, status, series_sort AS sort").
		Where("series_id = ?", series.ID).
		Order("series_sort ASC, id ASC").
		Scan(&posts).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"list": posts})
}

// AdminReorderSeriesPosts PUT /api/v1/admin/series/:id/posts —— 批量调整序号
func AdminReorderSeriesPosts(c *gin.Context) {
	id, err := strconvUint(c.Param("id"))
	if err != nil {
		common.Fail(c, common.CodeParamError, "专题 id 不合法")
		return
	}
	var req struct {
		Items []struct {
			PostID uint `json:"postId"`
			Sort   int  `json:"sort"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	var series model.Series
	if err := model.DB.First(&series, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "专题不存在")
		return
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if err := tx.Model(&model.Post{}).
				Where("id = ? AND series_id = ?", item.PostID, series.ID).
				Update("series_sort", item.Sort).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}
