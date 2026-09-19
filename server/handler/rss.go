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

// RSS GET /rss（根路径）—— RSS 2.0，最近 20 篇已发布文章，链接基于 settings.siteUrl
func RSS(c *gin.Context) {
	settings := getSettingsDTO(model.DB)
	siteURL := strings.TrimRight(settings.SiteURL, "/")

	var posts []model.Post
	if err := model.DB.Where("status = ?", model.PostPublished).
		Order("published_at DESC, id DESC").Limit(20).Find(&posts).Error; err != nil {
		common.ServerError(c, err)
		return
	}

	type rssItem struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`
		GUID        string `xml:"guid"`
	}
	type rssChannel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		LastBuild   string    `xml:"lastBuildDate"`
		Items       []rssItem `xml:"item"`
	}
	type rssDoc struct {
		XMLName xml.Name   `xml:"rss"`
		Version string     `xml:"version,attr"`
		Channel rssChannel `xml:"channel"`
	}

	items := make([]rssItem, 0, len(posts))
	for i := range posts {
		p := &posts[i]
		link := siteURL + "/posts/" + p.Slug
		pub := p.CreatedAt
		if p.PublishedAt != nil {
			pub = *p.PublishedAt
		}
		items = append(items, rssItem{
			Title:       p.Title,
			Link:        link,
			Description: p.Summary,
			PubDate:     pub.Format(time.RFC1123Z),
			GUID:        link,
		})
	}

	channelTitle := settings.SiteName
	if channelTitle == "" {
		channelTitle = "My Blog"
	}
	doc := rssDoc{
		Version: "2.0",
		Channel: rssChannel{
			Title:       channelTitle,
			Link:        siteURL,
			Description: settings.SiteDescription,
			LastBuild:   time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		common.ServerError(c, err)
		return
	}
	body := append([]byte(xml.Header), out...)
	c.Data(http.StatusOK, "application/rss+xml; charset=utf-8", body)
}
