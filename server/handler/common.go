// Package handler 实现全部 API（公开 + 管理端），博客规模业务简单，直接 GORM 不分 service 层。
package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/config"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	uploadDir string
	jwtSecret string
)

// SetConfig 由 router.Setup 注入运行配置
func SetConfig(cfg *config.Config) {
	uploadDir = cfg.UploadDir
	jwtSecret = cfg.JWTSecret
}

// ---------- 通用查询助手 ----------

// likeEscapeClause 所有 LIKE 均显式声明转义符，防通配符注入
const likeEscapeClause = "LIKE ? ESCAPE '\\'"

// escapeLike 转义 LIKE 通配符 % _ \
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func likeContains(s string) string {
	return "%" + escapeLike(s) + "%"
}

// queryUint 解析 query 中的非负整数，非法或缺失返回 0
func queryUint(c *gin.Context, key string) uint {
	v, err := strconv.ParseUint(c.Query(key), 10, 64)
	if err != nil {
		return 0
	}
	return uint(v)
}

// resolvePublishedPost :slug 参数同时接受数字 id 或 slug，仅返回已发布文章。
// 第二个返回值 false 表示不存在或未发布（调用方统一返回 10004）。
func resolvePublishedPost(param string) (*model.Post, bool, error) {
	var post model.Post
	if isAllDigits(param) {
		err := model.DB.Preload("Category").Preload("Tags").
			Where("id = ? AND status = ?", param, model.PostPublished).
			First(&post).Error
		if err == nil {
			return &post, true, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, err
		}
	}
	err := model.DB.Preload("Category").Preload("Tags").
		Where("slug = ? AND status = ?", param, model.PostPublished).
		First(&post).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &post, true, nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// postTitleMap 批量取文章标题（评论列表用）
func postTitleMap(postIDs []uint) map[uint]string {
	titles := make(map[uint]string, len(postIDs))
	if len(postIDs) == 0 {
		return titles
	}
	var posts []model.Post
	if err := model.DB.Select("id", "title").Where("id IN ?", postIDs).Find(&posts).Error; err != nil {
		return titles
	}
	for i := range posts {
		titles[posts[i].ID] = posts[i].Title
	}
	return titles
}

// ---------- slug 生成（契约：slug 空/重复自动生成 post-{id} 或追加 -id） ----------

func slugTaken(db *gorm.DB, slug string, excludeID uint) bool {
	var n int64
	db.Model(&model.Post{}).Where("slug = ? AND id <> ?", slug, excludeID).Count(&n)
	return n > 0
}

// ensurePostSlug 计算并落库文章最终 slug：
// 期望为空 → post-{id}；重复 → 追加 -{id}；仍重复 → post-{id}；再冲突 → 追加随机后缀。
func ensurePostSlug(db *gorm.DB, post *model.Post, desired string) error {
	final := strings.TrimSpace(desired)
	if final == "" {
		final = fmt.Sprintf("post-%d", post.ID)
	} else if slugTaken(db, final, post.ID) {
		final = fmt.Sprintf("%s-%d", final, post.ID)
	}
	if slugTaken(db, final, post.ID) {
		final = fmt.Sprintf("post-%d", post.ID)
	}
	if slugTaken(db, final, post.ID) {
		final = fmt.Sprintf("post-%d-%s", post.ID, randomHex(4))
	}
	post.Slug = final
	return db.Model(post).Update("slug", final).Error
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- 标签 upsert（契约：tags 按 name upsert） ----------

// upsertTagsByName 按名称查找或创建标签，去重、忽略空串
func upsertTagsByName(db *gorm.DB, names []string) ([]model.Tag, error) {
	tags := make([]model.Tag, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		if utf8.RuneCountInString(name) > 64 {
			return nil, fmt.Errorf("标签名称过长：%s", name)
		}
		var err error
		var tag model.Tag
		tag, err = upsertOneTag(db, name)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// upsertOneTag 查找或创建单个标签，并在名称无法派生 slug 时回退 tag-{id}
func upsertOneTag(db *gorm.DB, name string) (model.Tag, error) {
	var tag model.Tag
	err := db.Where("name = ?", name).First(&tag).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return tag, err
	}
	tag = model.Tag{Name: name, Slug: common.Slugify(name)}
	if err := db.Create(&tag).Error; err != nil {
		return tag, err
	}
	if tag.Slug == "" {
		tag.Slug = fmt.Sprintf("tag-%d", tag.ID)
		if err := db.Model(&tag).Update("slug", tag.Slug).Error; err != nil {
			return tag, err
		}
	}
	return tag, nil
}

// ---------- 分类/标签 slug 兜底 ----------

// taxonomySlug 计算分类/标签 slug：期望值优先，其次由名称派生，均不可用返回空串
func taxonomySlug(desired, name string) string {
	if s := strings.TrimSpace(desired); s != "" {
		return s
	}
	return common.Slugify(name)
}

func publishNowIfFirst(post *model.Post, status int8, updates map[string]any) {
	if status == model.PostPublished && post.PublishedAt == nil {
		updates["published_at"] = time.Now()
	}
}
