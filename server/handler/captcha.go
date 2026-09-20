package handler

// 图形验证码（契约 #86 / #11）：登录前校验，单次有效，5 分钟过期。
// 内存存储适配单管理员单实例部署；SVG 生成无第三方依赖。

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"myblog/server/common"

	"github.com/gin-gonic/gin"
)

const (
	captchaTTL      = 5 * time.Minute
	captchaCharset  = "23456789abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ"
	captchaLen      = 4
	captchaStoreCap = 5000
)

type captchaEntry struct {
	code     string
	expireAt time.Time
}

var (
	captchaMu    sync.Mutex
	captchaStore = make(map[string]captchaEntry)
)

func captchaRandInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

// purgeExpiredCaptcha 清理过期与超量条目（调用方需持锁）
func purgeExpiredCaptcha() {
	now := time.Now()
	for k, e := range captchaStore {
		if now.After(e.expireAt) {
			delete(captchaStore, k)
		}
	}
	// 防滥用：仍超量则丢弃最早过期的条目
	for len(captchaStore) >= captchaStoreCap {
		oldestKey, oldest := "", time.Time{}
		for k, e := range captchaStore {
			if oldest.IsZero() || e.expireAt.Before(oldest) {
				oldestKey, oldest = k, e.expireAt
			}
		}
		if oldestKey == "" {
			break
		}
		delete(captchaStore, oldestKey)
	}
}

// newCaptcha 生成一組验证码，返回 id、答案与 SVG 图片
func newCaptcha() (id, code, svg string) {
	cb := make([]byte, captchaLen)
	for i := range cb {
		cb[i] = captchaCharset[captchaRandInt(len(captchaCharset))]
	}
	code = string(cb)

	raw := make([]byte, 16)
	_, _ = rand.Read(raw)
	id = fmt.Sprintf("%x", raw)

	palette := []string{"#1554ad", "#0b6e99", "#3a5fcd", "#6f42c1", "#0d7a5f"}
	w, h := 132, 44
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, w, h, w, h)
	b.WriteString(`<rect width="100%" height="100%" fill="#eef2f7"/>`)
	// 干扰线
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1" opacity="0.5"/>`,
			captchaRandInt(w), captchaRandInt(h), captchaRandInt(w), captchaRandInt(h),
			palette[captchaRandInt(len(palette))])
	}
	// 字符（轻微旋转与上下偏移）
	for i := 0; i < captchaLen; i++ {
		x := 14 + i*28 + captchaRandInt(4)
		y := 30 + captchaRandInt(6) - 3
		rot := captchaRandInt(41) - 20
		color := palette[captchaRandInt(len(palette))]
		fmt.Fprintf(&b, `<text x="%d" y="%d" transform="rotate(%d %d %d)" font-family="Georgia, serif" font-size="27" font-weight="700" fill="%s" font-style="italic">%c</text>`,
			x, y, rot, x, y, color, code[i])
	}
	// 干扰点
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="1.2" fill="%s" opacity="0.45"/>`,
			captchaRandInt(w), captchaRandInt(h), palette[captchaRandInt(len(palette))])
	}
	b.WriteString(`</svg>`)
	svg = b.String()

	captchaMu.Lock()
	purgeExpiredCaptcha()
	captchaStore[id] = captchaEntry{code: code, expireAt: time.Now().Add(captchaTTL)}
	captchaMu.Unlock()
	return id, code, svg
}

// captchaVerifyResult 校验结果：区分「题目不存在/已过期」与「答案不匹配」。
// 两者对外同为 10001，但提示文案不同——否则用户分不清是「手滑填错」还是
// 「题目真的作废了」，只会反复看到同一句「已过期」而误判为一直刷新一直过期。
type captchaVerifyResult int

const (
	captchaOK       captchaVerifyResult = iota
	captchaNotFound                     // 从未签发 / 已被消费 / 已过期
	captchaMismatch                     // 题目有效但答案不符
)

// verifyCaptcha 校验并销毁（无论对错都单次有效）
func verifyCaptcha(id, input string) captchaVerifyResult {
	if id == "" || input == "" {
		return captchaNotFound
	}
	captchaMu.Lock()
	defer captchaMu.Unlock()
	e, ok := captchaStore[id]
	if !ok {
		return captchaNotFound
	}
	delete(captchaStore, id)
	if time.Now().After(e.expireAt) {
		return captchaNotFound
	}
	if !strings.EqualFold(strings.TrimSpace(e.code), strings.TrimSpace(input)) {
		return captchaMismatch
	}
	return captchaOK
}

// GetCaptchaAnswer 读取指定验证码答案——仅供同进程测试使用（无 HTTP 暴露）。
// 读取不消费；登录校验仍是单次有效。
func GetCaptchaAnswer(id string) string {
	captchaMu.Lock()
	defer captchaMu.Unlock()
	return captchaStore[id].code
}

// AdminGetCaptcha GET /api/v1/admin/auth/captcha —— 无需登录（登录前置）
func AdminGetCaptcha(c *gin.Context) {
	// 验证码是「一次性票据」：一旦被任何中间层缓存，客户端就会拿到
	// 已被消费/过期的 captchaId，表现为「一直刷新一直过期」。
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	id, _, svg := newCaptcha()
	common.OK(c, gin.H{"captchaId": id, "image": svg})
}
