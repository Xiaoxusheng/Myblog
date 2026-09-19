package handler

// 安全防护管理端接口（契约「安全防护」）：封禁列表 / 手动封禁 / 解封 / 防护事件查询。
// 封禁本体与拦截逻辑在 middleware.Guard；本文件只做参数校验、审计与响应包装。

import (
	"errors"
	"net"
	"strings"
	"time"

	"myblog/server/common"
	"myblog/server/middleware"

	"github.com/gin-gonic/gin"
)

const (
	manualBanMaxMinutes = 60 * 24 * 365 // 手动封禁时长上限：1 年
	banReasonMaxRunes   = 200
)

// AdminListBans GET /api/v1/admin/security/bans —— 生效中的封禁列表
func AdminListBans(c *gin.Context) {
	g := middleware.CurrentGuard()
	if g == nil {
		common.OK(c, gin.H{"list": []middleware.BanInfo{}})
		return
	}
	common.OK(c, gin.H{"list": g.ListBans()})
}

// AdminCreateBan POST /api/v1/admin/security/bans —— 手动封禁
// body {ip, durationMinutes?, reason?}；durationMinutes 缺省/0 = 永久
func AdminCreateBan(c *gin.Context) {
	var req struct {
		IP              string `json:"ip"`
		DurationMinutes *int   `json:"durationMinutes"`
		Reason          string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, "参数错误")
		return
	}
	ip := strings.TrimSpace(req.IP)
	if net.ParseIP(ip) == nil {
		common.Fail(c, common.CodeParamError, "ip 不是合法的 IP 地址")
		return
	}

	g := middleware.CurrentGuard()
	if g == nil {
		common.ServerError(c, errors.New("防护模块未初始化"))
		return
	}
	// 防自锁：白名单 IP 与当前登录 IP 禁止封禁
	if g.IsWhitelisted(ip) {
		common.Fail(c, common.CodeParamError, "该 IP 在防护白名单中，禁止封禁")
		return
	}
	if ip == c.ClientIP() {
		common.Fail(c, common.CodeParamError, "不能封禁当前登录 IP，否则会将自己锁在门外")
		return
	}

	duration := time.Duration(0) // 永久
	if req.DurationMinutes != nil {
		if *req.DurationMinutes < 0 || *req.DurationMinutes > manualBanMaxMinutes {
			common.Fail(c, common.CodeParamError, "durationMinutes 取值范围为 0（永久）到 525600（一年）")
			return
		}
		duration = time.Duration(*req.DurationMinutes) * time.Minute
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "管理员手动封禁"
	}
	if r := []rune(reason); len(r) > banReasonMaxRunes {
		reason = string(r[:banReasonMaxRunes])
	}

	g.Ban(ip, reason, duration)
	writeAudit(c, "security.ban", "security", ip, "手动封禁 IP "+ip+"："+reason)
	common.OK(c, gin.H{"list": g.ListBans()})
}

// AdminDeleteBan DELETE /api/v1/admin/security/bans/:ip —— 解除封禁
func AdminDeleteBan(c *gin.Context) {
	ip := c.Param("ip")
	if net.ParseIP(ip) == nil {
		common.Fail(c, common.CodeParamError, "ip 不是合法的 IP 地址")
		return
	}
	g := middleware.CurrentGuard()
	if g == nil {
		common.ServerError(c, errors.New("防护模块未初始化"))
		return
	}
	if !g.Unban(ip) {
		common.Fail(c, common.CodeNotFound, "该 IP 不在封禁名单中")
		return
	}
	writeAudit(c, "security.unban", "security", ip, "解除 IP 封禁："+ip)
	common.OK(c, gin.H{"list": g.ListBans()})
}

// AdminDefenseEvents GET /api/v1/admin/security/events?limit= —— 最近防护事件（进程内存，重启清零）
func AdminDefenseEvents(c *gin.Context) {
	limit := int(queryUint(c, "limit"))
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	g := middleware.CurrentGuard()
	if g == nil {
		common.OK(c, gin.H{"list": []middleware.DefenseEvent{}})
		return
	}
	common.OK(c, gin.H{"list": g.ListEvents(limit)})
}
