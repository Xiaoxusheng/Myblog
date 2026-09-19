package handler

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// Sitemap GET /sitemap.xml（根路径）—— sitemap 0.9，包含首页、文章、自定义页面、分类、标签，链接基于 settings.siteUrl
func Sitemap(c *gin.Context) {
	settings := getSettingsDTO(model.DB)
	siteURL := strings.TrimRight(settings.SiteURL, "/")
	if siteURL == "" {
		siteURL = "http://" + c.Request.Host
	}

	type entry struct {
		Loc        string  `xml:"loc"`
		Lastmod    *string `xml:"lastmod,omitempty"`
		Changefreq string  `xml:"changefreq,omitempty"`
	}
	type urlset struct {
		XMLName xml.Name `xml:"urlset"`
		Xmlns   string   `xml:"xmlns,attr"`
		Urls    []entry  `xml:"url"`
	}

	now := time.Now().Format("2006-01-02")
	urls := []entry{
		{Loc: siteURL + "/", Lastmod: &now, Changefreq: "daily"},
	}

	var posts []model.Post
	if err := model.DB.Where("status = ?", model.PostPublished).
		Order("published_at DESC").Find(&posts).Error; err == nil {
		for i := range posts {
			p := &posts[i]
			loc := siteURL + "/posts/" + p.Slug
			if p.PublishedAt != nil {
				t := p.PublishedAt.Format("2006-01-02")
				urls = append(urls, entry{Loc: loc, Lastmod: &t, Changefreq: "weekly"})
			} else {
				urls = append(urls, entry{Loc: loc, Changefreq: "weekly"})
			}
		}
	}

	var pages []model.Page
	if err := model.DB.Where("status = ?", 1).Find(&pages).Error; err == nil {
		for i := range pages {
			urls = append(urls, entry{Loc: siteURL + "/pages/" + pages[i].Slug, Changefreq: "monthly"})
		}
	}

	var categories []model.Category
	if err := model.DB.Find(&categories).Error; err == nil {
		for i := range categories {
			urls = append(urls, entry{Loc: siteURL + "/category/" + categories[i].Slug, Changefreq: "weekly"})
		}
	}

	var tags []model.Tag
	if err := model.DB.Find(&tags).Error; err == nil {
		for i := range tags {
			urls = append(urls, entry{Loc: siteURL + "/tag/" + tags[i].Slug, Changefreq: "weekly"})
		}
	}

	out, err := xml.MarshalIndent(urlset{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Urls:  urls,
	}, "", "  ")
	if err != nil {
		common.ServerError(c, err)
		return
	}
	body := append([]byte(xml.Header), out...)
	c.Data(http.StatusOK, "application/xml; charset=utf-8", body)
}

// Robots GET /robots.txt（根路径）—— 允许抓取，声明 Sitemap 地址
func Robots(c *gin.Context) {
	settings := getSettingsDTO(model.DB)
	siteURL := strings.TrimRight(settings.SiteURL, "/")
	if siteURL == "" {
		siteURL = "http://" + c.Request.Host
	}
	body := "User-agent: *\nAllow: /\nDisallow: /api/\nDisallow: /admin\n\nSitemap: " + siteURL + "/sitemap.xml\n"
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(body))
}
