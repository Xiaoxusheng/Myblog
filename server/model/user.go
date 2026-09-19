package model

import "time"

// User 管理员（单管理员模式，users 表仅一条记录）
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"type:varchar(64);uniqueIndex" json:"username"`
	Password  string    `gorm:"type:varchar(128)" json:"-"` // bcrypt，不外泄
	Nickname  string    `gorm:"type:varchar(64)" json:"nickname"`
	Email     string    `gorm:"type:varchar(128)" json:"email"`
	Avatar    string    `gorm:"type:varchar(512)" json:"avatar"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UserDTO 契约 User 对象（登录 / me / profile 返回）
type UserDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// DTO 转换为契约对象
func (u *User) DTO() UserDTO {
	return UserDTO{
		ID:       u.ID,
		Username: u.Username,
		Nickname: u.Nickname,
		Email:    u.Email,
		Avatar:   u.Avatar,
	}
}
