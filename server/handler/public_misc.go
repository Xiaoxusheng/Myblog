package handler

import (
	"sort"
	"time"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// Archive GET /api/v1/archive —— 按年分组倒序，仅已发布
func Archive(c *gin.Context) {
	var posts []model.Post
	if err := model.DB.Select("id", "title", "slug", "created_at", "published_at").
		Where("status = ?", model.PostPublished).
		Order("published_at DESC, id DESC").Find(&posts).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	type archiveItem struct {
		ID        uint      `json:"id"`
		Title     string    `json:"title"`
		Slug      string    `json:"slug"`
		CreatedAt time.Time `json:"createdAt"`
	}
	type archiveYear struct {
		Year  int           `json:"year"`
		Items []archiveItem `json:"items"`
	}

	yearMap := make(map[int]*archiveYear)
	years := make([]int, 0, 4)
	for i := range posts {
		t := posts[i].CreatedAt
		if posts[i].PublishedAt != nil {
			t = *posts[i].PublishedAt
		}
		year := t.Year()
		y, ok := yearMap[year]
		if !ok {
			y = &archiveYear{Year: year, Items: []archiveItem{}}
			yearMap[year] = y
			years = append(years, year)
		}
		y.Items = append(y.Items, archiveItem{
			ID:        posts[i].ID,
			Title:     posts[i].Title,
			Slug:      posts[i].Slug,
			CreatedAt: posts[i].CreatedAt,
		})
	}
	sort.Slice(years, func(i, j int) bool { return years[i] > years[j] })

	list := make([]archiveYear, 0, len(years))
	for _, y := range years {
		list = append(list, *yearMap[y])
	}
	common.OK(c, list)
}

// GetPage GET /api/v1/pages/:slug —— 已发布自定义页面（仅 status=1 可见；
// 草稿/隐藏/定时发布一律 10004，未发布页面的预览走管理端 #105）
func GetPage(c *gin.Context) {
	var page model.Page
	err := model.DB.Where("slug = ? AND status = ?", c.Param("slug"), model.PostPublished).First(&page).Error
	if err != nil {
		common.Fail(c, common.CodeNotFound, "页面不存在")
		return
	}
	common.OK(c, gin.H{"page": model.ToPageDTO(&page)})
}

// ListLinks GET /api/v1/links —— 仅 visible，按 sort 升序
func ListLinks(c *gin.Context) {
	var links []model.Link
	if err := model.DB.Where("visible = ?", true).
		Order("sort ASC, id ASC").Find(&links).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"list": links})
}
