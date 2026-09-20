package handler

import (
	"strings"
	"time"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// ---------- 时间线（契约 #91-94 管理侧 / #99 公开侧） ----------

// timelinePostRef 时间线节点关联的文章摘要
type timelinePostRef struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Slug   string `json:"slug"`
	Status int8   `json:"status,omitempty"` // 仅管理侧返回
}

// assembleTimelinePosts 批量补齐节点的关联文章（一次查询）；publishedOnly 时仅携带已发布文章
func assembleTimelinePosts(events []model.TimelineEvent, publishedOnly bool) []gin.H {
	ids := make([]uint, 0, len(events))
	seen := make(map[uint]bool, len(events))
	for _, ev := range events {
		if ev.PostID > 0 && !seen[ev.PostID] {
			seen[ev.PostID] = true
			ids = append(ids, ev.PostID)
		}
	}
	postMap := make(map[uint]timelinePostRef, len(ids))
	if len(ids) > 0 {
		q := model.DB.Select("id", "title", "slug", "status").Where("id IN ?", ids)
		if publishedOnly {
			q = q.Where("status = ?", model.PostPublished)
		}
		var posts []model.Post
		if err := q.Find(&posts).Error; err == nil {
			for _, p := range posts {
				ref := timelinePostRef{ID: p.ID, Title: p.Title, Slug: p.Slug}
				if !publishedOnly {
					ref.Status = p.Status
				}
				postMap[p.ID] = ref
			}
		}
	}
	list := make([]gin.H, 0, len(events))
	for _, ev := range events {
		var post *timelinePostRef
		if ref, ok := postMap[ev.PostID]; ok {
			post = &ref
		}
		list = append(list, gin.H{
			"id":          ev.ID,
			"title":       ev.Title,
			"content":     ev.Content,
			"eventDate":   ev.EventDate,
			"image":       ev.Image,
			"postId":      ev.PostID,
			"post":        post,
			"projectName": ev.ProjectName,
			"projectUrl":  ev.ProjectURL,
			"visible":     ev.Visible,
			"sort":        ev.Sort,
			"createdAt":   ev.CreatedAt,
			"updatedAt":   ev.UpdatedAt,
		})
	}
	return list
}

// timelineOrder 契约排序：sort 升序（默认 0，负值置顶）→ 日期倒序 → id 倒序
const timelineOrder = "sort ASC, event_date DESC, id DESC"

// AdminListTimeline GET /api/v1/admin/timeline
func AdminListTimeline(c *gin.Context) {
	pq := common.ParsePage(c, 20)

	var total int64
	if err := model.DB.Model(&model.TimelineEvent{}).Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var events []model.TimelineEvent
	if err := model.DB.Order(timelineOrder).
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&events).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, pq.Data(assembleTimelinePosts(events, false), total))
}

// timelinePayload 创建/更新时间线节点入参
type timelinePayload struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	EventDate   string `json:"eventDate"` // RFC3339
	Image       string `json:"image"`
	PostID      uint   `json:"postId"`
	ProjectName string `json:"projectName"`
	ProjectURL  string `json:"projectUrl"`
	Visible     bool   `json:"visible"`
	Sort        int    `json:"sort"`
}

// parse 校验并解析字段；失败时已写入错误响应
func (p *timelinePayload) parse(c *gin.Context) (time.Time, bool) {
	title := strings.TrimSpace(p.Title)
	switch {
	case title == "":
		common.Fail(c, common.CodeParamError, "标题不能为空")
		return time.Time{}, false
	case utf8.RuneCountInString(title) > 100:
		common.Fail(c, common.CodeParamError, "标题不能超过 100 字")
		return time.Time{}, false
	case utf8.RuneCountInString(p.Content) > 5000:
		common.Fail(c, common.CodeParamError, "内容不能超过 5000 字")
		return time.Time{}, false
	case utf8.RuneCountInString(p.Image) > 512 || utf8.RuneCountInString(p.ProjectURL) > 512 ||
		utf8.RuneCountInString(p.ProjectName) > 100:
		common.Fail(c, common.CodeParamError, "字段长度超出限制")
		return time.Time{}, false
	}
	eventDate, err := time.Parse(time.RFC3339, strings.TrimSpace(p.EventDate))
	if err != nil {
		common.Fail(c, common.CodeParamError, "日期格式错误，须为 RFC3339")
		return time.Time{}, false
	}
	if p.PostID > 0 {
		var count int64
		if err := model.DB.Model(&model.Post{}).Where("id = ?", p.PostID).Count(&count).Error; err != nil || count == 0 {
			common.Fail(c, common.CodeParamError, "关联文章不存在")
			return time.Time{}, false
		}
	}
	return eventDate, true
}

func (p *timelinePayload) apply(ev *model.TimelineEvent, eventDate time.Time) {
	ev.Title = strings.TrimSpace(p.Title)
	ev.Content = p.Content
	ev.EventDate = eventDate
	ev.Image = strings.TrimSpace(p.Image)
	ev.PostID = p.PostID
	ev.ProjectName = strings.TrimSpace(p.ProjectName)
	ev.ProjectURL = strings.TrimSpace(p.ProjectURL)
	ev.Visible = p.Visible
	ev.Sort = p.Sort
}

// AdminCreateTimeline POST /api/v1/admin/timeline
func AdminCreateTimeline(c *gin.Context) {
	var req timelinePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	eventDate, ok := req.parse(c)
	if !ok {
		return
	}
	ev := &model.TimelineEvent{}
	req.apply(ev, eventDate)
	if err := model.DB.Create(ev).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"item": assembleTimelinePosts([]model.TimelineEvent{*ev}, false)[0]})
}

// AdminUpdateTimeline PUT /api/v1/admin/timeline/:id
func AdminUpdateTimeline(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req timelinePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	eventDate, ok := req.parse(c)
	if !ok {
		return
	}
	var ev model.TimelineEvent
	if err := model.DB.First(&ev, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "时间线节点不存在")
		return
	}
	req.apply(&ev, eventDate)
	if err := model.DB.Save(&ev).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"item": assembleTimelinePosts([]model.TimelineEvent{ev}, false)[0]})
}

// AdminDeleteTimeline DELETE /api/v1/admin/timeline/:id
func AdminDeleteTimeline(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var ev model.TimelineEvent
	if err := model.DB.First(&ev, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "时间线节点不存在")
		return
	}
	if err := model.DB.Delete(&ev).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// ListTimeline GET /api/v1/timeline —— 公开：仅 visible，post 仅已发布
func ListTimeline(c *gin.Context) {
	var events []model.TimelineEvent
	if err := model.DB.Where("visible = ?", true).
		Order(timelineOrder).Limit(200).Find(&events).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"list": assembleTimelinePosts(events, true)})
}
