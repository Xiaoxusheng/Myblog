package handler

// llms.txt（AEO，契约 #83）：面向 AI 检索的站点说明与内容索引。
// 只输出可验证的真实数据（站点设置 + 已发布内容），不做编造的元数据。

import (
	"fmt"
	"net/http"
	"strings"

	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// LlmsTxt GET /llms.txt（根路径）
func LlmsTxt(c *gin.Context) {
	settings := getSettingsDTO(model.DB)
	siteURL := strings.TrimRight(settings.SiteURL, "/")
	if siteURL == "" {
		siteURL = "http://" + c.Request.Host
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", settings.SiteName)
	if strings.TrimSpace(settings.SiteDescription) != "" {
		fmt.Fprintf(&b, "> %s\n\n", settings.SiteDescription)
	}

	// 核心页面
	b.WriteString("## 核心页面\n\n")
	fmt.Fprintf(&b, "- [首页](%s/)：最新文章\n", siteURL)
	fmt.Fprintf(&b, "- [归档](%s/archives)：按时间浏览全部文章\n", siteURL)
	fmt.Fprintf(&b, "- [专题](%s/series)：系列教程\n", siteURL)
	fmt.Fprintf(&b, "- [友链](%s/links)\n", siteURL)

	// 分类
	var categories []model.Category
	_ = model.DB.Order("id ASC").Find(&categories).Error
	if len(categories) > 0 {
		b.WriteString("\n## 分类\n\n")
		for _, cat := range categories {
			fmt.Fprintf(&b, "- [%s](%s/category/%s)\n", cat.Name, siteURL, cat.Slug)
		}
	}

	// 专题（仅可见）
	var series []model.Series
	_ = model.DB.Where("visible = ?", true).Order("sort ASC, id ASC").Find(&series).Error
	if len(series) > 0 {
		b.WriteString("\n## 专题\n\n")
		for _, s := range series {
			desc := strings.TrimSpace(s.Description)
			if desc != "" {
				fmt.Fprintf(&b, "- [%s](%s/series/%s)：%s\n", s.Name, siteURL, s.Slug, desc)
			} else {
				fmt.Fprintf(&b, "- [%s](%s/series/%s)\n", s.Name, siteURL, s.Slug)
			}
		}
	}

	// 文章索引（仅已发布，按发布时间倒序，含摘要）
	var posts []model.Post
	_ = model.DB.Where("status = ?", model.PostPublished).
		Order("published_at DESC, id DESC").Limit(200).Find(&posts).Error
	if len(posts) > 0 {
		b.WriteString("\n## 文章\n\n")
		for _, p := range posts {
			summary := strings.TrimSpace(p.Summary)
			if summary != "" {
				if r := []rune(summary); len(r) > 120 {
					summary = string(r[:120]) + "…"
				}
				fmt.Fprintf(&b, "- [%s](%s/post/%s)：%s\n", p.Title, siteURL, p.Slug, summary)
			} else {
				fmt.Fprintf(&b, "- [%s](%s/post/%s)\n", p.Title, siteURL, p.Slug)
			}
		}
	}

	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(b.String()))
}
