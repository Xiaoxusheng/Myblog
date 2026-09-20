package handler

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// ---------- 版本发布记录（契约 #95-98 管理侧 / #100 公开侧） ----------

// versionPattern SemVer 兼容：去 v 前缀后三段数字 + 可选预发布段（不含构建元数据）
var versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-[0-9A-Za-z.-]+)?$`)

// normalizeVersion 校验并归一版本号：去 v 前缀，统一存储形态；非法返回 false
func normalizeVersion(raw string) (string, bool) {
	v := strings.TrimSpace(raw)
	if v == "" || utf8.RuneCountInString(v) > 32 || !versionPattern.MatchString(v) {
		return "", false
	}
	return strings.TrimPrefix(v, "v"), true
}

// changelogOrder 契约排序：sort 升序（默认 0）→ 发布日期倒序 → id 倒序（默认即版本倒序）
const changelogOrder = "sort ASC, released_at DESC, id DESC"

// AdminListChangelogs GET /api/v1/admin/changelogs?status=
func AdminListChangelogs(c *gin.Context) {
	pq := common.ParsePage(c, 20)

	query := model.DB.Model(&model.Changelog{})
	if s := c.Query("status"); s != "" {
		status, err := strconv.Atoi(s)
		if err != nil || status < 0 || status > 1 {
			common.Fail(c, common.CodeParamError, "status 仅允许 0 或 1")
			return
		}
		query = query.Where("status = ?", int8(status))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	var logs []model.Changelog
	if err := query.Order(changelogOrder).
		Offset(pq.Offset()).Limit(pq.PageSize).Find(&logs).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, pq.Data(logs, total))
}

// changelogPayload 创建/更新版本记录入参
type changelogPayload struct {
	Version    string `json:"version"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	ReleasedAt string `json:"releasedAt"` // RFC3339
	Status     int8   `json:"status"`
	Sort       int    `json:"sort"`
}

// parse 校验并解析字段；失败时已写入错误响应
func (p *changelogPayload) parse(c *gin.Context) (string, time.Time, bool) {
	version, ok := normalizeVersion(p.Version)
	if !ok {
		common.Fail(c, common.CodeParamError, "版本号格式错误，须兼容 SemVer（如 1.2.0、v1.2.0-beta.1）")
		return "", time.Time{}, false
	}
	switch {
	case utf8.RuneCountInString(strings.TrimSpace(p.Title)) > 100:
		common.Fail(c, common.CodeParamError, "标题不能超过 100 字")
		return "", time.Time{}, false
	case utf8.RuneCountInString(p.Content) > 20000:
		common.Fail(c, common.CodeParamError, "内容不能超过 20000 字")
		return "", time.Time{}, false
	case p.Status != int8(model.ChangelogDraft) && p.Status != int8(model.ChangelogPublished):
		common.Fail(c, common.CodeParamError, "status 仅允许 0 或 1")
		return "", time.Time{}, false
	}
	releasedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(p.ReleasedAt))
	if err != nil {
		common.Fail(c, common.CodeParamError, "发布日期格式错误，须为 RFC3339")
		return "", time.Time{}, false
	}
	return version, releasedAt, true
}

// versionTaken 检查版本号唯一性（excludeID>0 时排除自身）
func versionTaken(version string, excludeID uint) bool {
	var count int64
	q := model.DB.Model(&model.Changelog{}).Where("version = ?", version)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

// AdminCreateChangelog POST /api/v1/admin/changelogs
func AdminCreateChangelog(c *gin.Context) {
	var req changelogPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	version, releasedAt, ok := req.parse(c)
	if !ok {
		return
	}
	if versionTaken(version, 0) {
		common.Fail(c, common.CodeParamError, "版本号已存在")
		return
	}
	log := model.Changelog{
		Version:    version,
		Title:      strings.TrimSpace(req.Title),
		Content:    req.Content,
		ReleasedAt: releasedAt,
		Status:     req.Status,
		Sort:       req.Sort,
	}
	if err := model.DB.Create(&log).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"item": log})
}

// AdminUpdateChangelog PUT /api/v1/admin/changelogs/:id
func AdminUpdateChangelog(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req changelogPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	version, releasedAt, ok := req.parse(c)
	if !ok {
		return
	}
	var log model.Changelog
	if err := model.DB.First(&log, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "版本记录不存在")
		return
	}
	if versionTaken(version, log.ID) {
		common.Fail(c, common.CodeParamError, "版本号已存在")
		return
	}
	log.Version = version
	log.Title = strings.TrimSpace(req.Title)
	log.Content = req.Content
	log.ReleasedAt = releasedAt
	log.Status = req.Status
	log.Sort = req.Sort
	if err := model.DB.Save(&log).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"item": log})
}

// AdminDeleteChangelog DELETE /api/v1/admin/changelogs/:id?force=
// 已发布记录带删除保护：未携带 force=true 时拒绝
func AdminDeleteChangelog(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var log model.Changelog
	if err := model.DB.First(&log, id).Error; err != nil {
		common.Fail(c, common.CodeNotFound, "版本记录不存在")
		return
	}
	if log.Status == model.ChangelogPublished && c.Query("force") != "true" {
		common.Fail(c, common.CodeParamError, "已发布的版本记录需二次确认后删除")
		return
	}
	if err := model.DB.Delete(&log).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// ListChangelogs GET /api/v1/changelog —— 公开：仅已发布，≤200 条
func ListChangelogs(c *gin.Context) {
	var logs []model.Changelog
	if err := model.DB.Where("status = ?", model.ChangelogPublished).
		Order(changelogOrder).Limit(200).Find(&logs).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"list": logs})
}
