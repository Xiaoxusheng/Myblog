package handler

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"myblog/server/common"
	"myblog/server/middleware"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// dummyBcryptHash 用户名不存在时也执行一次同代价的 bcrypt 比较，
// 避免通过响应耗时探测用户名是否存在（时序侧信道）。
var dummyBcryptHash = []byte("$2a$10$vi1hqIQLtqQA8HlMXdgy2.fPx8zDY96I8P7l6IFQufxmFYP8W1ilq")

// Login POST /api/v1/admin/auth/login —— remember=true 签发 7 天，否则 24h；失败 20001
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	if strings.TrimSpace(req.Username) == "" || req.Password == "" {
		common.Fail(c, common.CodeParamError, "用户名和密码不能为空")
		return
	}

	var user model.User
	err := model.DB.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(req.Password)) // 时序对齐，结果无意义
		middleware.MarkLoginFailure(c)
		common.Fail(c, common.CodeLoginFailed, "用户名或密码错误")
		return
	}
	if err != nil {
		common.ServerError(c, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		middleware.MarkLoginFailure(c)
		common.Fail(c, common.CodeLoginFailed, "用户名或密码错误")
		return
	}

	middleware.MarkLoginSuccess(c)
	ttl := 24 * time.Hour
	if req.Remember {
		ttl = 7 * 24 * time.Hour
	}
	token, err := middleware.GenerateToken(user.ID, user.Username, jwtSecret, ttl)
	if err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"token": token, "user": user.DTO()})
}

// currentUser 从 JWT 上下文解析当前管理员
func currentUser(c *gin.Context) (*model.User, bool) {
	uid, ok := c.Get(middleware.ContextUserID)
	if !ok {
		return nil, false
	}
	id, ok := uid.(uint)
	if !ok {
		return nil, false
	}
	var user model.User
	if err := model.DB.First(&user, id).Error; err != nil {
		return nil, false
	}
	return &user, true
}

// Me GET /api/v1/admin/auth/me
func Me(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		common.Unauthorized(c)
		return
	}
	common.OK(c, gin.H{"user": user.DTO()})
}

// UpdatePassword PUT /api/v1/admin/auth/password —— 新密码 ≥6 位；旧密码错误 20001
func UpdatePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	user, ok := currentUser(c)
	if !ok {
		common.Unauthorized(c)
		return
	}
	if utf8.RuneCountInString(req.NewPassword) < 6 {
		common.Fail(c, common.CodeParamError, "新密码至少 6 位")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)) != nil {
		common.Fail(c, common.CodeLoginFailed, "原密码错误")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		common.ServerError(c, err)
		return
	}
	if err := model.DB.Model(user).Update("password", string(hash)).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, nil)
}

// UpdateProfile PUT /api/v1/admin/auth/profile
func UpdateProfile(c *gin.Context) {
	var req struct {
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	user, ok := currentUser(c)
	if !ok {
		common.Unauthorized(c)
		return
	}

	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Email = strings.TrimSpace(req.Email)
	switch {
	case req.Nickname == "":
		common.Fail(c, common.CodeParamError, "昵称不能为空")
		return
	case utf8.RuneCountInString(req.Nickname) > 64:
		common.Fail(c, common.CodeParamError, "昵称不能超过 64 字")
		return
	case utf8.RuneCountInString(req.Email) > 128:
		common.Fail(c, common.CodeParamError, "邮箱过长")
		return
	case utf8.RuneCountInString(req.Avatar) > 512:
		common.Fail(c, common.CodeParamError, "头像地址过长")
		return
	}

	// struct + Select 更新：走 GORM serializer，email 透明加密落库
	//（map 更新不触发 serializer，会绕过加密）
	updates := model.User{Nickname: req.Nickname, Email: req.Email, Avatar: req.Avatar}
	if err := model.DB.Model(user).Select("nickname", "email", "avatar").Updates(updates).Error; err != nil {
		common.ServerError(c, err)
		return
	}
	common.OK(c, gin.H{"user": user.DTO()})
}
