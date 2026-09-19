package model

// 文章版本历史（契约 #50-52 / database.md post_revisions）：
// 每次保存内容真正变化才生成快照；恢复操作先快照当前内容再应用目标版本，保证恢复可撤销。

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// PostRevision 文章版本快照（不含标签；status 仅作保存时状态的历史记录，恢复时不回写）
type PostRevision struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	PostID     uint   `gorm:"uniqueIndex:idx_post_version" json:"postId"`
	Version    int    `gorm:"uniqueIndex:idx_post_version" json:"version"`
	Title      string `gorm:"type:varchar(200)" json:"title"`
	Slug       string `gorm:"type:varchar(200)" json:"slug"`
	Summary    string `gorm:"type:varchar(1000)" json:"summary"`
	Cover      string `gorm:"type:varchar(512)" json:"cover"`
	Content    string `gorm:"type:text" json:"content"`
	CategoryID uint   `json:"categoryId"`
	IsTop      bool   `json:"isTop"`
	Status     int8   `json:"status"`
	// Remark 变更说明：首次保存 / 修改标题、正文 / 恢复前快照 / 恢复自 v3
	Remark    string    `gorm:"type:varchar(200)" json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
}

// RevisionAutoThrottle auto 保存的版本生成防抖间隔：距最新版本不足该时长时仅保存内容不生成版本
const RevisionAutoThrottle = 120 * time.Second

// PostSnapshot 版本比较用的文章字段集合（与版本生成规则一一对应）
type PostSnapshot struct {
	Title      string
	Slug       string
	Summary    string
	Cover      string
	Content    string
	CategoryID uint
	IsTop      bool
	Status     int8
}

// SnapshotOf 提取文章的版本快照字段
func SnapshotOf(p *Post) PostSnapshot {
	return PostSnapshot{
		Title:      p.Title,
		Slug:       p.Slug,
		Summary:    p.Summary,
		Cover:      p.Cover,
		Content:    p.Content,
		CategoryID: p.CategoryID,
		IsTop:      p.IsTop,
		Status:     p.Status,
	}
}

// Snapshot 版本记录 → 快照字段
func (r *PostRevision) Snapshot() PostSnapshot {
	return PostSnapshot{
		Title:      r.Title,
		Slug:       r.Slug,
		Summary:    r.Summary,
		Cover:      r.Cover,
		Content:    r.Content,
		CategoryID: r.CategoryID,
		IsTop:      r.IsTop,
		Status:     r.Status,
	}
}

// SnapshotEqual 判断两个快照是否一致
func SnapshotEqual(a, b PostSnapshot) bool {
	return a == b
}

// LatestPostRevision 返回文章最新版本；无版本（历史存量文章）返回 (nil, nil)
func LatestPostRevision(db *gorm.DB, postID uint) (*PostRevision, error) {
	var rev PostRevision
	err := db.Where("post_id = ?", postID).Order("version DESC").First(&rev).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

// CreatePostRevision 以 post 当前内容写入该文章的下一个版本
func CreatePostRevision(db *gorm.DB, post *Post, remark string) error {
	var maxVersion int
	if err := db.Model(&PostRevision{}).Where("post_id = ?", post.ID).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVersion).Error; err != nil {
		return err
	}
	rev := PostRevision{
		PostID:     post.ID,
		Version:    maxVersion + 1,
		Title:      post.Title,
		Slug:       post.Slug,
		Summary:    post.Summary,
		Cover:      post.Cover,
		Content:    post.Content,
		CategoryID: post.CategoryID,
		IsTop:      post.IsTop,
		Status:     post.Status,
		Remark:     remark,
	}
	return db.Create(&rev).Error
}

// RevisionRemark 对比新状态与最新版本生成变更说明（如「修改标题、正文」）；
// 与最新版本一致返回空串；文章尚无任何版本时返回「首次保存」。
func RevisionRemark(next *Post, prev *PostRevision) string {
	if prev == nil {
		return "首次保存"
	}
	var parts []string
	if next.Title != prev.Title {
		parts = append(parts, "标题")
	}
	if next.Content != prev.Content {
		parts = append(parts, "正文")
	}
	if next.Summary != prev.Summary {
		parts = append(parts, "摘要")
	}
	if next.Cover != prev.Cover {
		parts = append(parts, "封面")
	}
	if next.CategoryID != prev.CategoryID {
		parts = append(parts, "分类")
	}
	if next.IsTop != prev.IsTop {
		parts = append(parts, "置顶")
	}
	if next.Status != prev.Status {
		parts = append(parts, "状态")
	}
	if next.Slug != prev.Slug {
		parts = append(parts, "Slug")
	}
	if len(parts) == 0 {
		return ""
	}
	return "修改" + strings.Join(parts, "、")
}
