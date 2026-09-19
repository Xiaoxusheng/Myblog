package model

// PageView 访问统计（契约 #66-68）。
// 隐私边界：不存明文 IP（仅 SHA-256 哈希）、不存 Cookie/Authorization/表单内容；
// UA 解析结果（device/browser/os）入库供聚合，原始 UA 仅留诊断。

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"
)

type PageView struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Path          string    `gorm:"type:varchar(512)" json:"path"`
	PostID        uint      `gorm:"index" json:"postId"` // 0=非文章页
	VisitorHash   string    `gorm:"index;type:varchar(64)" json:"visitorHash"`
	IPHash        string    `gorm:"type:varchar(64)" json:"ipHash"`
	UserAgent     string    `gorm:"type:varchar(512)" json:"userAgent"`
	Referer       string    `gorm:"type:varchar(512)" json:"referer"`
	RefererSource string    `gorm:"type:varchar(16)" json:"refererSource"` // direct/search/github/social/other
	DeviceType    string    `gorm:"type:varchar(16)" json:"deviceType"`    // desktop/mobile/tablet
	Browser       string    `gorm:"type:varchar(32)" json:"browser"`
	OS            string    `gorm:"type:varchar(32)" json:"os"`
	CreatedAt     time.Time `gorm:"index" json:"createdAt"`
}

// hashSecret 哈希盐：优先加密密钥，否则 JWT 密钥（由 handler 注入，见 SetHashSecret）
var hashSecret string

// SetHashSecret 由 router.Setup 注入哈希盐（敏感字段密钥或 JWT 密钥）
func SetHashSecret(secret string) {
	hashSecret = secret
}

func hashWithSalt(input string) string {
	sum := sha256.Sum256([]byte(input + "|" + hashSecret))
	return hex.EncodeToString(sum[:])
}

// NewPageView 由原始请求要素构造统计行（解析 UA / referer / 计算哈希）
func NewPageView(path string, postID uint, ua, referer, ip string) PageView {
	if len(ua) > 512 {
		ua = ua[:512]
	}
	if len(referer) > 512 {
		referer = referer[:512]
	}
	if len(path) > 512 {
		path = path[:512]
	}
	return PageView{
		Path:          path,
		PostID:        postID,
		VisitorHash:   hashWithSalt(ip + "|" + ua),
		IPHash:        hashWithSalt(ip),
		UserAgent:     ua,
		Referer:       referer,
		RefererSource: classifyReferer(referer),
		DeviceType:    parseDeviceType(ua),
		Browser:       parseBrowser(ua),
		OS:            parseOS(ua),
		CreatedAt:     time.Now(),
	}
}

// classifyReferer referer 归类（可靠分类，不做猜测）：direct/search/github/social/other
func classifyReferer(referer string) string {
	if strings.TrimSpace(referer) == "" {
		return "direct"
	}
	host := ""
	if u, err := url.Parse(referer); err == nil {
		host = strings.ToLower(u.Hostname())
	}
	if host == "" {
		return "other"
	}
	switch {
	case host == "github.com" || strings.HasSuffix(host, ".github.com"):
		return "github"
	case containsAny(host,
		"google.", "bing.com", "baidu.com", "sogou.com", "so.com", "duckduckgo.com", "yandex.", "sm.cn"):
		return "search"
	case containsAny(host,
		"twitter.com", "x.com", "t.co", "weibo.com", "facebook.com", "reddit.com",
		"zhihu.com", "juejin.cn", "v2ex.com", "sspai.com", "bilibili.com", "t.me"):
		return "social"
	default:
		return "other"
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// parseDeviceType 设备类型：平板优先于手机判断（Android 无 Mobi 视为平板）
func parseDeviceType(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case containsAny(ua, "ipad", "tablet"):
		return "tablet"
	case containsAny(ua, "android") && !strings.Contains(ua, "mobi"):
		return "tablet"
	case containsAny(ua, "mobi", "iphone", "android"):
		return "mobile"
	default:
		return "desktop"
	}
}

// parseBrowser 浏览器粗解析（顺序即优先级：Edge/Opera 内核串先于 Chrome/Safari）
func parseBrowser(ua string) string {
	switch {
	case strings.Contains(ua, "Edg/"), strings.Contains(ua, "Edge/"):
		return "Edge"
	case strings.Contains(ua, "OPR/"), strings.Contains(ua, "Opera"):
		return "Opera"
	case strings.Contains(ua, "Firefox/"):
		return "Firefox"
	case strings.Contains(ua, "Chrome/"), strings.Contains(ua, "CriOS/"):
		return "Chrome"
	case strings.Contains(ua, "Safari/"):
		return "Safari"
	case strings.Contains(ua, "MSIE"), strings.Contains(ua, "Trident/"):
		return "IE"
	default:
		return "Other"
	}
}

// parseOS 操作系统粗解析
func parseOS(ua string) string {
	switch {
	case strings.Contains(ua, "Windows NT"), strings.Contains(ua, "Windows Phone"):
		return "Windows"
	case strings.Contains(ua, "Mac OS X"), strings.Contains(ua, "Macintosh"):
		return "macOS"
	case strings.Contains(ua, "Android"):
		return "Android"
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"), strings.Contains(ua, "iOS"):
		return "iOS"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	default:
		return "Other"
	}
}
