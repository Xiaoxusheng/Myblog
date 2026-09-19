package common

import "strings"

// Slugify 由名称生成 URL 友好的 slug：
// 小写、空白折叠为连字符、仅保留 [a-z0-9-_]；
// 中文等无法转换的名称返回空串，由调用方回退（如 tag-{id}）。
func Slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := true // 抑制首部连字符
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_':
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		default:
			// 丢弃其他字符（含中文、空格）
		}
	}
	return strings.Trim(b.String(), "-")
}
