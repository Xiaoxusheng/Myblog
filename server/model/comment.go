package model

import "time"

// 评论状态（契约枚举）
const (
	CommentPending  int8 = 0 // 待审核
	CommentApproved int8 = 1 // 已通过
	CommentRejected int8 = 2 // 已拒绝
)

// Comment 评论（两级树：parent_id=0 顶级，其余为回复）
type Comment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PostID    uint      `gorm:"index" json:"postId"`
	ParentID  uint      `gorm:"index;default:0" json:"parentId"`
	Nickname  string    `gorm:"type:varchar(64)" json:"nickname"`
	Email     string    `gorm:"type:varchar(128)" json:"email"` // 不对公开侧返回
	Website   string    `gorm:"type:varchar(256)" json:"website"`
	Content   string    `gorm:"type:text" json:"content"`
	Status    int8      `gorm:"index;default:0" json:"status"`
	IP        string    `gorm:"type:varchar(64)" json:"ip"` // 仅管理端可见
	IsAdmin   bool      `json:"isAdmin"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CommentPublicDTO 公开评论（契约 CommentPublic，仅已通过；两级树）
type CommentPublicDTO struct {
	ID        uint               `json:"id"`
	ParentID  uint               `json:"parentId"`
	Nickname  string             `json:"nickname"`
	Website   string             `json:"website"`
	Content   string             `json:"content"`
	IsAdmin   bool               `json:"isAdmin"`
	CreatedAt time.Time          `json:"createdAt"`
	Children  []CommentPublicDTO `json:"children"`
}

// CommentAdminDTO 管理端评论（契约 CommentAdmin）
type CommentAdminDTO struct {
	ID        uint      `json:"id"`
	PostID    uint      `json:"postId"`
	PostTitle string    `json:"postTitle"`
	ParentID  uint      `json:"parentId"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email"`
	Website   string    `json:"website"`
	Content   string    `json:"content"`
	Status    int8      `json:"status"`
	IP        string    `json:"ip"`
	IsAdmin   bool      `json:"isAdmin"`
	CreatedAt time.Time `json:"createdAt"`
}
