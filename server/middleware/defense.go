package middleware

// 防护 Guard：反爬 / 恶意请求拦截 / 自动封 IP（契约「安全防护」）。
// 三层检查按序执行：封禁名单 → WAF（UA / 路径 / 查询串规则）→ 全局限流；
// 违规按次计点（strikes），滑动窗口内达到阈值自动封禁。
// 封禁持久化到 banned_ips 表，重启后由 InitGuard 重新加载；计点与事件仅存内存。

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"myblog/server/common"
	"myblog/server/config"
	"myblog/server/model"

	"github.com/gin-gonic/gin"
)

// ---------- 对外数据结构 ----------

// BanInfo 封禁条目（管理端展示）
type BanInfo struct {
	IP        string     `json:"ip"`
	Reason    string     `json:"reason"`
	Source    string     `json:"source"` // auto（自动）/ manual（手动）
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt"` // nil = 永久
}

// DefenseEvent 防护事件（管理端展示，仅内存环形缓冲，重启清零）
type DefenseEvent struct {
	Time   time.Time `json:"time"`
	IP     string    `json:"ip"`
	Kind   string    `json:"kind"` // waf / rate_limit / login_block / auto_ban / manual_ban / unban
	Detail string    `json:"detail"`
}

// 事件类型常量
const (
	eventWAF        = "waf"
	eventRateLimit  = "rate_limit"
	eventLoginBlock = "login_block"
	eventAutoBan    = "auto_ban"
	eventManualBan  = "manual_ban"
	eventUnban      = "unban"
)

// WAF 模式
const (
	wafOff   = "off"
	wafLog   = "log"
	wafBlock = "block"
)

const (
	defenseEventCap = 300  // 事件环形缓冲容量
	sweepStrikesAt  = 8192 // 计点表超过该值触发全量清理
	sweepBansAt     = 4096 // 封禁表超过该值触发全量清理
)

// ---------- 内部状态 ----------

type banEntry struct {
	Reason    string
	Source    string
	CreatedAt time.Time
	ExpiresAt time.Time // 零值 = 永久
}

type strikeLog struct {
	times []time.Time
}

type rateWindow struct {
	start time.Time
	count int
}

// Guard 防护实例；全部共享状态由 mu / rateMu 保护
type Guard struct {
	mu        sync.Mutex
	bans      map[string]*banEntry
	strikes   map[string]*strikeLog
	events    []DefenseEvent
	whitelist []*net.IPNet

	rateMu       sync.Mutex
	rateCounters map[string]*rateWindow

	rateLimit    int
	wafMode      string
	banThreshold int
	banWindow    time.Duration
	banDuration  time.Duration
}

// guard 当前防护实例（router.Setup 时由 InitGuard 赋值）
var guard *Guard

// InitGuard 构建防护实例并从数据库加载生效中的封禁；router.Setup 调用
func InitGuard(cfg *config.Config) *Guard {
	g := &Guard{
		bans:         make(map[string]*banEntry),
		strikes:      make(map[string]*strikeLog),
		rateCounters: make(map[string]*rateWindow),
		rateLimit:    cfg.RateLimitPerMin,
		wafMode:      cfg.WAFMode,
		banThreshold: cfg.BanThreshold,
		banWindow:    cfg.BanWindow,
		banDuration:  cfg.BanDuration,
		whitelist:    parseWhitelist(cfg.BanWhitelist),
	}
	if g.banWindow <= 0 {
		g.banWindow = 10 * time.Minute
	}
	if g.banDuration <= 0 {
		g.banDuration = 30 * time.Minute
	}
	guard = g
	g.loadBansFromDB()
	return g
}

// CurrentGuard 返回当前防护实例（未初始化返回 nil）
func CurrentGuard() *Guard { return guard }

// ResetGuard 仅供测试用例之间隔离防护状态
func ResetGuard() { guard = nil }

// parseWhitelist 解析白名单：IP 归一为 /32 或 /128 网段，统一用 Contains 匹配
func parseWhitelist(items []string) []*net.IPNet {
	var out []*net.IPNet
	for _, item := range items {
		if _, cidr, err := net.ParseCIDR(item); err == nil {
			out = append(out, cidr)
			continue
		}
		if ip := net.ParseIP(item); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			out = append(out, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		log.Printf("[defense] 白名单项 %q 不是合法 IP/CIDR，已忽略", item)
	}
	return out
}

// ---------- 中间件 ----------

// Middleware 防护中间件：白名单 → 封禁 → WAF → 限流，命中即短路。
// 白名单 IP 完全跳过（放行自身测试、监控探针等可信来源）。
func (g *Guard) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if g == nil {
			c.Next()
			return
		}
		ip := c.ClientIP()
		if g.IsWhitelisted(ip) {
			c.Next()
			return
		}

		// 1. 封禁名单：直接 403，不再计点/记录（避免日志被攻击者刷爆）
		if _, banned := g.isBanned(ip); banned {
			c.AbortWithStatusJSON(http.StatusForbidden,
				common.Response{Code: common.CodeForbidden, Message: "请求被拒绝", Data: nil})
			return
		}

		// 2. WAF：恶意 UA / 探测路径 / 恶意查询串
		if g.wafMode == wafBlock || g.wafMode == wafLog {
			if detail, hit := wafCheck(c); hit {
				if g.wafMode == wafBlock {
					g.addEvent(ip, eventWAF, detail)
					g.addStrike(ip, "WAF："+detail)
					c.AbortWithStatusJSON(http.StatusForbidden,
						common.Response{Code: common.CodeForbidden, Message: "请求被拒绝", Data: nil})
					return
				}
				// log 模式：仅记录，不计点不拦截，便于上线前评估误报
				g.addEvent(ip, eventWAF, detail)
			}
		}

		// 3. 全局限流：固定窗口计数，超限 429 + Retry-After，并计点
		if g.rateLimit > 0 {
			if retryAfter, allowed := g.allow(ip); !allowed {
				g.addEvent(ip, eventRateLimit, "超过全局限流 "+strconv.Itoa(g.rateLimit)+" 次/分钟")
				g.addStrike(ip, "超过全局限流")
				c.Header("Retry-After", strconv.Itoa(retryAfter))
				c.AbortWithStatusJSON(http.StatusTooManyRequests,
					common.Response{Code: common.CodeTooFrequent, Message: "请求过于频繁，请稍后再试", Data: nil})
				return
			}
		}
		c.Next()
	}
}

// ---------- 封禁 ----------

// isBanned 命中返回 true；发现已过期条目顺手删除（懒清理）
func (g *Guard) isBanned(ip string) (*banEntry, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	e, ok := g.bans[ip]
	if !ok {
		return nil, false
	}
	if !e.ExpiresAt.IsZero() && !time.Now().Before(e.ExpiresAt) {
		delete(g.bans, ip)
		return nil, false
	}
	return e, true
}

// IsWhitelisted 判断 IP 是否命中白名单（IP 或 CIDR）
func (g *Guard) IsWhitelisted(ip string) bool {
	if g == nil || len(g.whitelist) == 0 {
		return false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range g.whitelist {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

// Ban 手动封禁；duration<=0 表示永久。返回封禁后的条目快照。
func (g *Guard) Ban(ip, reason string, duration time.Duration) BanInfo {
	now := time.Now()
	entry := &banEntry{Reason: reason, Source: "manual", CreatedAt: now}
	var expiresAt *time.Time
	if duration > 0 {
		entry.ExpiresAt = now.Add(duration)
		expiresAt = &entry.ExpiresAt
	}
	g.mu.Lock()
	g.bans[ip] = entry
	delete(g.strikes, ip) // 封禁后历史计点作废
	g.addEventLocked(ip, eventManualBan, reason)
	g.sweepLocked(now)
	g.mu.Unlock()
	g.persistBan(ip, reason, "manual", expiresAt)
	return snapshotBan(ip, entry)
}

// Unban 解除封禁；返回该 IP 此前是否在封禁中
func (g *Guard) Unban(ip string) bool {
	g.mu.Lock()
	_, existed := g.bans[ip]
	delete(g.bans, ip)
	if existed {
		g.addEventLocked(ip, eventUnban, "解除封禁")
	}
	g.mu.Unlock()
	if existed {
		g.persistUnban(ip)
	}
	return existed
}

// ListBans 返回生效中的封禁（新创建的在前）
func (g *Guard) ListBans() []BanInfo {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	list := make([]BanInfo, 0, len(g.bans))
	for ip, e := range g.bans {
		if !e.ExpiresAt.IsZero() && !now.Before(e.ExpiresAt) {
			continue
		}
		list = append(list, snapshotBan(ip, e))
	}
	return list
}

func snapshotBan(ip string, e *banEntry) BanInfo {
	info := BanInfo{
		IP:        ip,
		Reason:    e.Reason,
		Source:    e.Source,
		CreatedAt: e.CreatedAt,
	}
	if !e.ExpiresAt.IsZero() {
		exp := e.ExpiresAt
		info.ExpiresAt = &exp
	}
	return info
}

// addStrike 记一次违规；滑动窗口内达到阈值自动封禁
func (g *Guard) addStrike(ip, reason string) {
	if g.banThreshold <= 0 {
		return // 自动封禁关闭：计点无意义，省内存
	}
	now := time.Now()
	autoBanned := false
	g.mu.Lock()
	st := g.strikes[ip]
	if st == nil {
		st = &strikeLog{}
		g.strikes[ip] = st
	}
	kept := st.times[:0]
	for _, ts := range st.times { // 剔除窗口外的历史计点
		if now.Sub(ts) < g.banWindow {
			kept = append(kept, ts)
		}
	}
	st.times = append(kept, now)
	if len(st.times) >= g.banThreshold {
		count := len(st.times)
		entry := &banEntry{
			Reason:    reason,
			Source:    "auto",
			CreatedAt: now,
			ExpiresAt: now.Add(g.banDuration),
		}
		g.bans[ip] = entry
		delete(g.strikes, ip)
		g.addEventLocked(ip, eventAutoBan, reason+"（窗口内违规 "+strconv.Itoa(count)+" 次）")
		autoBanned = true
	}
	g.sweepLocked(now)
	g.mu.Unlock()
	if autoBanned {
		exp := now.Add(g.banDuration)
		g.persistBan(ip, reason, "auto", &exp)
	}
}

// sweepLocked 全量清理过期条目，防 map 无界增长（调用方需持有 mu）
func (g *Guard) sweepLocked(now time.Time) {
	if len(g.bans) > sweepBansAt {
		for ip, e := range g.bans {
			if !e.ExpiresAt.IsZero() && !now.Before(e.ExpiresAt) {
				delete(g.bans, ip)
			}
		}
	}
	if len(g.strikes) > sweepStrikesAt {
		for ip, st := range g.strikes {
			latest := st.times[len(st.times)-1]
			if now.Sub(latest) >= g.banWindow {
				delete(g.strikes, ip)
			}
		}
	}
}

func (g *Guard) persistBan(ip, reason, source string, expiresAt *time.Time) {
	if model.DB == nil {
		return
	}
	if err := model.UpsertBan(model.DB, ip, reason, source, expiresAt); err != nil {
		log.Printf("[defense] 封禁 %s 持久化失败：%v", ip, err)
	}
}

func (g *Guard) persistUnban(ip string) {
	if model.DB == nil {
		return
	}
	if err := model.DeleteBan(model.DB, ip); err != nil {
		log.Printf("[defense] 解除封禁 %s 持久化失败：%v", ip, err)
	}
}

// loadBansFromDB 启动加载生效中的封禁；数据库未就绪或为空时静默跳过
func (g *Guard) loadBansFromDB() {
	if model.DB == nil {
		return
	}
	list, err := model.LoadActiveBans(model.DB)
	if err != nil {
		log.Printf("[defense] 加载封禁名单失败（防护仍以内存状态运行）：%v", err)
		return
	}
	for _, b := range list {
		entry := &banEntry{Reason: b.Reason, Source: b.Source, CreatedAt: b.CreatedAt}
		if b.ExpiresAt != nil {
			entry.ExpiresAt = *b.ExpiresAt
		}
		g.bans[b.IP] = entry
	}
	if len(list) > 0 {
		log.Printf("[defense] 已从数据库加载 %d 条生效中的 IP 封禁", len(list))
	}
}

// ---------- 限流 ----------

// allow 固定窗口限流；返回 (Retry-After 秒数, 是否放行)
func (g *Guard) allow(ip string) (int, bool) {
	g.rateMu.Lock()
	defer g.rateMu.Unlock()
	now := time.Now()
	if len(g.rateCounters) > sweepStrikesAt {
		for k, w := range g.rateCounters {
			if now.Sub(w.start) >= time.Minute {
				delete(g.rateCounters, k)
			}
		}
	}
	w, ok := g.rateCounters[ip]
	if !ok || now.Sub(w.start) >= time.Minute {
		g.rateCounters[ip] = &rateWindow{start: now, count: 1}
		return 0, true
	}
	if w.count >= g.rateLimit {
		remain := time.Minute - now.Sub(w.start)
		retryAfter := int(remain/time.Second) + 1
		return retryAfter, false
	}
	w.count++
	return 0, true
}

// ---------- 事件 ----------

// addEvent 记录一条防护事件；容量固定，超出丢弃最旧
func (g *Guard) addEvent(ip, kind, detail string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.addEventLocked(ip, kind, detail)
}

func (g *Guard) addEventLocked(ip, kind, detail string) {
	g.events = append(g.events, DefenseEvent{
		Time:   time.Now(),
		IP:     ip,
		Kind:   kind,
		Detail: detail,
	})
	if len(g.events) > defenseEventCap {
		g.events = g.events[len(g.events)-defenseEventCap:]
	}
}

// ListEvents 返回最近 n 条事件（新→旧）
func (g *Guard) ListEvents(n int) []DefenseEvent {
	g.mu.Lock()
	defer g.mu.Unlock()
	if n <= 0 || n > len(g.events) {
		n = len(g.events)
	}
	out := make([]DefenseEvent, 0, n)
	for i := len(g.events) - 1; i >= len(g.events)-n; i-- {
		out = append(out, g.events[i])
	}
	return out
}

// ---------- WAF 规则 ----------

// 规则取舍原则：宁缺毋滥——只收录对该博客几乎不可能误报的特征。
// curl / wget / python-requests / Feedly 等常见工具与阅读器不在拦截之列，由全局限流约束。

// wafToolUA 已知攻击/扫描工具的 UA 特征（小写包含匹配）
var wafToolUA = []string{
	"sqlmap", "nikto", "nuclei", "masscan", "zgrab", "nmap",
	"dirbuster", "dirb", "gobuster", "wfuzz", "ffuf", "feroxbuster",
	"acunetix", "nessus", "openvas", "wpscan", "havij", "sqlninja", "hydra",
}

// wafPathContains 路径包含特征（CMS/组件探测，小写包含匹配）
var wafPathContains = []string{
	"wp-admin", "wp-login", "wp-content", "wp-includes", "xmlrpc",
	"phpmyadmin", "phpunit", "admin.php", "login.php", "setup.php",
}

// wafPathPrefixes 路径前缀特征（隐藏文件/中间件探测）
var wafPathPrefixes = []string{
	"/.env", "/.git", "/.aws", "/.ssh", "/.svn", "/cgi-bin/", "/actuator",
}

// wafPathSuffixes 路径后缀特征（脚本文件探测；本服务无任何动态脚本文件）
var wafPathSuffixes = []string{
	".php", ".asp", ".aspx", ".jsp", ".cgi", ".env",
}

// wafQueryInjection 查询串注入特征（URL 解码后小写包含匹配；搜索接口豁免）
var wafQueryInjection = []string{
	"union select", "union all select", "information_schema",
	"load_file(", "benchmark(", "waitfor delay", "sleep(",
	"or 1=1", "' or 1", "<script", "javascript:", "onerror=", "onload=",
}

// wafQueryTraversal 目录穿越特征（所有路径检查，含搜索关键词）
var wafQueryTraversal = []string{
	"../", "..\\",
}

// wafCheck 依次检查 UA / 路径 / 查询串，命中返回 (特征说明, true)
func wafCheck(c *gin.Context) (string, bool) {
	// 1. User-Agent：空 UA 或攻击工具
	ua := strings.ToLower(c.Request.UserAgent())
	if ua == "" {
		return "空 User-Agent", true
	}
	for _, tool := range wafToolUA {
		if strings.Contains(ua, tool) {
			return "攻击工具 UA：" + c.Request.UserAgent(), true
		}
	}

	// 2. 路径：CMS 探测 / 隐藏文件 / 脚本后缀
	path := strings.ToLower(c.Request.URL.Path)
	for _, p := range wafPathContains {
		if strings.Contains(path, p) {
			return "探测路径 " + c.Request.URL.Path, true
		}
	}
	for _, p := range wafPathPrefixes {
		if strings.HasPrefix(path, p) {
			return "探测路径 " + c.Request.URL.Path, true
		}
	}
	for _, p := range wafPathSuffixes {
		if strings.HasSuffix(path, p) {
			return "探测路径 " + c.Request.URL.Path, true
		}
	}

	// 3. 查询串：先做目录穿越（所有接口），再做注入特征（搜索接口豁免——
	// 技术博客检索 "union select" 等关键词是正常使用）
	raw := strings.ToLower(c.Request.URL.RawQuery)
	if raw == "" {
		return "", false
	}
	decoded := raw
	if dec, err := url.QueryUnescape(raw); err == nil {
		decoded = dec
	}
	for _, p := range wafQueryTraversal {
		if strings.Contains(decoded, p) {
			return "查询串目录穿越：" + c.Request.URL.RawQuery, true
		}
	}
	if isSearchPath(c.Request.URL.Path) {
		return "", false
	}
	for _, p := range wafQueryInjection {
		if strings.Contains(decoded, p) {
			return "查询串注入特征：" + c.Request.URL.RawQuery, true
		}
	}
	return "", false
}

// isSearchPath 关键词搜索接口：查询串注入特征检查豁免（目录穿越仍检查）
func isSearchPath(path string) bool {
	return path == "/api/v1/posts" || path == "/api/v1/admin/posts"
}
