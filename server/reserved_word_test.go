package main

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"myblog/server/model"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 本文件防御的是「MySQL 8 保留字未加反引号」这一类 bug。
//
// 背景：notifications.read 与 settings.key 都是 MySQL 8 保留字。
// 裸写会生成 `... WHERE read = ?` / `... WHERE key = ?` → Error 1064 语法错误。
// **SQLite 不保留这两个词，所以本机 Go 测试全绿也不能证明线上没问题** ——
// 线上就是这么炸的：管理端铃铛每 60s 轮询 /admin/notifications 报 500。
//
// 用例策略：挂一个 GORM logger 钩子，把**真实 HTTP 请求路径上执行过的 SQL** 抓下来，
// 再断言语句里没有裸写的保留字标识符。这样断言的是生产代码本身，
// 而不是用例里手写的字符串 —— 把源码改回裸写，用例必须失败。

// ---------- SQL 捕获 ----------

// sqlCapture 记录经过 GORM 的所有 SQL 语句
type sqlCapture struct {
	logger.Interface
	sqls []string
}

func (c *sqlCapture) LogMode(logger.LogLevel) logger.Interface { return c }

func (c *sqlCapture) Info(ctx context.Context, msg string, data ...interface{}) {}

func (c *sqlCapture) Warn(ctx context.Context, msg string, data ...interface{}) {}

func (c *sqlCapture) Error(ctx context.Context, msg string, data ...interface{}) {}

// Trace 是唯一会拿到最终 SQL 的回调
func (c *sqlCapture) Trace(ctx context.Context, begin time.Time,
	fc func() (string, int64), err error) {
	sql, _ := fc()
	c.sqls = append(c.sqls, sql)
}

// withCapture 在 t.Cleanup 期间把 model.DB 换成带 SQL 捕获的会话
func withCapture(t *testing.T) *sqlCapture {
	t.Helper()
	if model.DB == nil {
		t.Fatal("model.DB 未初始化")
	}
	cap := &sqlCapture{}
	old := model.DB
	model.DB = old.Session(&gorm.Session{Logger: cap})
	t.Cleanup(func() { model.DB = old })
	return cap
}

// findBare 在捕获到的语句里找 word 的裸写（标识符位置、未被反引号包裹）
func (c *sqlCapture) findBare(word string) (string, bool) {
	for _, sql := range c.sqls {
		if hasBareIdentifier(sql, word) {
			return sql, true
		}
	}
	return "", false
}

func (c *sqlCapture) dump() string {
	return strings.Join(c.sqls, " ||\n")
}

// hasBareIdentifier 判断 word 是否作为**未加反引号**的标识符出现。
// 判据：word 独立成词（前后非标识符字符），且紧邻两侧都不是反引号。
// 这样 `read` 判定为安全，WHERE read = ? 判定为危险。
func hasBareIdentifier(sql, word string) bool {
	low := strings.ToLower(sql)
	w := strings.ToLower(word)
	for i := 0; ; {
		j := strings.Index(low[i:], w)
		if j < 0 {
			return false
		}
		j += i
		before := byte(0)
		if j > 0 {
			before = low[j-1]
		}
		var after byte
		if j+len(w) < len(low) {
			after = low[j+len(w)]
		}
		// 词边界：前后不能是标识符字符（字母/数字/下划线/点）
		leftOK := !isIdentByte(before)
		rightOK := !isIdentByte(after)
		if leftOK && rightOK {
			// 被反引号包裹 = 安全
			if before != '`' && after != '`' {
				return true
			}
		}
		i = j + len(w)
	}
}

func isIdentByte(b byte) bool {
	return b == '_' || b == '.' ||
		(b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// ---------- 用例 ----------

// TestReservedWordReadNotBareInNotificationPaths
// 走真实 HTTP 接口 #31 / #32，断言 notifications 相关语句里 read 未被裸写。
func TestReservedWordReadNotBareInNotificationPaths(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	cap := withCapture(t)

	// #31 通知列表（线上报 500 的那条，含未读数统计）
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/notifications", token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("#31 通知列表失败：code=%d msg=%s", e.Code, e.Message)
	}
	// #32 全部标记已读
	rec = doJSON(t, r, http.MethodPut, "/api/v1/admin/notifications/read-all", token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("#32 全部已读失败：code=%d msg=%s", e.Code, e.Message)
	}

	if len(cap.sqls) == 0 {
		t.Fatal("未捕获到任何 SQL —— 捕获钩子失效，用例失去意义")
	}
	// 确认真的走到了 notifications 表（否则断言是在空集上通过）
	touched := false
	for _, s := range cap.sqls {
		if strings.Contains(strings.ToLower(s), "notifications") {
			touched = true
			break
		}
	}
	if !touched {
		t.Fatalf("未捕获到 notifications 相关语句，用例无效：\n%s", cap.dump())
	}
	if sql, bad := cap.findBare("read"); bad {
		t.Fatalf("MySQL 8 保留字 read 被裸写（线上会 Error 1064）：\n%s", sql)
	}
}

// TestReservedWordKeyNotBareInSettingPath
// 该语句的错误在 admin_setting.go 里被吞掉并回退默认值 ——
// 线上表现为「设置项改了不生效」，且日志里完全看不到，属于最隐蔽的一类。
func TestReservedWordKeyNotBareInSettingPath(t *testing.T) {
	r := newTestApp(t)
	token := loginToken(t, r)
	cap := withCapture(t)

	// 触发 slug 变更流程（会读 autoRedirectOnSlugChange 开关）+ 设置读写
	rec := doJSON(t, r, http.MethodGet, "/api/v1/admin/settings", token, nil)
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("读取设置失败：code=%d msg=%s", e.Code, e.Message)
	}
	post := createPost(t, r, token, map[string]any{"title": "改名测试", "slug": "before-slug"})
	rec = doJSON(t, r, http.MethodPut, fmt.Sprintf("/api/v1/admin/posts/%d", post.ID), token,
		map[string]any{
			"title": "改名测试", "slug": "after-slug", "summary": "s",
			"content": "c", "status": 1, "tags": []string{},
		})
	if e := decode(t, rec); e.Code != 0 {
		t.Fatalf("更新文章失败：code=%d msg=%s", e.Code, e.Message)
	}

	if len(cap.sqls) == 0 {
		t.Fatal("未捕获到任何 SQL —— 捕获钩子失效")
	}
	touched := false
	for _, s := range cap.sqls {
		if strings.Contains(strings.ToLower(s), "settings") {
			touched = true
			break
		}
	}
	if !touched {
		t.Fatalf("未捕获到 settings 相关语句，用例无效：\n%s", cap.dump())
	}
	if sql, bad := cap.findBare("key"); bad {
		// SELECT ... WHERE key = ? 这类；注意 ORDER BY `settings`.`key` 是带引号的
		t.Fatalf("MySQL 8 保留字 key 被裸写（线上 Error 1064 且错误被吞掉）：\n%s", sql)
	}
}

// TestReservedWordModelColumnsDeclared 「防新增」：
// 结构体字段命中保留字时必须显式声明 gorm:"column:xxx"
func TestReservedWordModelColumnsDeclared(t *testing.T) {
	cases := []struct {
		v     any
		field string
		col   string
	}{
		{&model.Notification{}, "Read", "read"},
		{&model.Setting{}, "Key", "key"},
	}
	for _, c := range cases {
		tag := fieldTag(c.v, c.field)
		if tag == "" {
			t.Errorf("%T.%s 未找到 gorm tag", c.v, c.field)
			continue
		}
		if !strings.Contains(tag, "column:"+c.col) {
			t.Errorf("%T.%s 的列名 %q 是 MySQL 8 保留字，必须显式声明 gorm:\"column:%s\"，当前 tag=%q",
				c.v, c.field, c.col, c.col, tag)
		}
	}
}

// ---------- 小工具 ----------

func fieldTag(v any, field string) string {
	rt := reflect.TypeOf(v)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	f, ok := rt.FieldByName(field)
	if !ok {
		return ""
	}
	return f.Tag.Get("gorm")
}
