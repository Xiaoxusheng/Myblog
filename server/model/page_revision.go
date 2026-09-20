package model

// 页面版本历史（契约 #101-103 / database.md page_revisions）：
// 与 PostRevision 同一范式——每次保存内容真正变化才生成快照；
// 恢复操作先快照当前内容再应用目标版本，保证恢复可撤销。

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// PageRevision 页面版本快照（status 仅作保存时状态的历史记录，恢复时不回写）
type PageRevision struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	PageID  uint   `gorm:"uniqueIndex:idx_page_version" json:"pageId"`
	Version int    `gorm:"uniqueIndex:idx_page_version" json:"version"`
	Title   string `gorm:"type:varchar(200)" json:"title"`
	Slug    string `gorm:"type:varchar(200)" json:"slug"`
	Content string `gorm:"type:text" json:"content"`
	Status  int8   `json:"status"`
	// Remark 变更说明：首次保存 / 修改标题、正文 / 恢复前快照 / 恢复自 v3
	Remark    string    `gorm:"type:varchar(200)" json:"remark"`
	CreatedAt time.Time `json:"createdAt"`
}

// PageSnapshot 版本比较用的页面字段集合（与版本生成规则一一对应）
type PageSnapshot struct {
	Title   string
	Slug    string
	Content string
	Status  int8
}

// PageSnapshotOf 提取页面的版本快照字段
func PageSnapshotOf(p *Page) PageSnapshot {
	return PageSnapshot{
		Title:   p.Title,
		Slug:    p.Slug,
		Content: p.Content,
		Status:  p.Status,
	}
}

// Snapshot 版本记录 → 快照字段
func (r *PageRevision) Snapshot() PageSnapshot {
	return PageSnapshot{
		Title:   r.Title,
		Slug:    r.Slug,
		Content: r.Content,
		Status:  r.Status,
	}
}

// PageSnapshotEqual 判断两个页面快照是否一致
func PageSnapshotEqual(a, b PageSnapshot) bool {
	return a == b
}

// LatestPageRevision 返回页面最新版本；无版本（历史存量页面）返回 (nil, nil)
func LatestPageRevision(db *gorm.DB, pageID uint) (*PageRevision, error) {
	var rev PageRevision
	err := db.Where("page_id = ?", pageID).Order("version DESC").First(&rev).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rev, nil
}

// CreatePageRevision 以 page 当前内容写入该页面的下一个版本
func CreatePageRevision(db *gorm.DB, page *Page, remark string) error {
	var maxVersion int
	if err := db.Model(&PageRevision{}).Where("page_id = ?", page.ID).
		Select("COALESCE(MAX(version), 0)").Scan(&maxVersion).Error; err != nil {
		return err
	}
	rev := PageRevision{
		PageID:  page.ID,
		Version: maxVersion + 1,
		Title:   page.Title,
		Slug:    page.Slug,
		Content: page.Content,
		Status:  page.Status,
		Remark:  remark,
	}
	return db.Create(&rev).Error
}

// PageRevisionRemark 对比新状态与最新版本生成变更说明（如「修改标题、正文」）；
// 与最新版本一致返回空串；页面尚无任何版本时返回「首次保存」。
func PageRevisionRemark(next *Page, prev *PageRevision) string {
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
