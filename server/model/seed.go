package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Seed 幂等初始化种子数据：仅在对应表为空 / 记录缺失时写入
func Seed(db *gorm.DB) error {
	if err := seedUser(db); err != nil {
		return err
	}
	if err := seedSettings(db); err != nil {
		return err
	}
	category, err := seedTaxonomy(db)
	if err != nil {
		return err
	}
	if err := seedPosts(db, category); err != nil {
		return err
	}
	if err := seedComments(db); err != nil {
		return err
	}
	seedLinks(db)
	seedPage(db)
	return nil
}

func seedUser(db *gorm.DB) error {
	var count int64
	if err := db.Model(&User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	admin := User{Username: "admin", Password: string(hash), Nickname: "管理员", Email: "admin@example.com"}
	if err := db.Create(&admin).Error; err != nil {
		return err
	}
	logSeed("已创建初始管理员 admin / admin123（请登录后尽快修改密码）")
	return nil
}

func seedSettings(db *gorm.DB) error {
	defaults := map[string]string{
		"siteName":        "My Blog",
		"siteDescription": "",
		"siteKeywords":    "",
		"siteUrl":         "",
		"logo":            "",
		"notice":          "",
		"icp":             "",
		"footerText":      "",
		"commentEnabled":  "true",
		"postPageSize":    "10",
	}
	for key, value := range defaults {
		var count int64
		if err := db.Model(&Setting{}).Where("`key` = ?", key).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&Setting{Key: key, Value: value}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedTaxonomy(db *gorm.DB) (*Category, error) {
	// 分类「未分类」必须存在
	var category Category
	err := db.Where("name = ?", "未分类").First(&category).Error
	if err != nil {
		category = Category{Name: "未分类", Slug: "uncategorized"}
		if err := db.Create(&category).Error; err != nil {
			return nil, err
		}
	}

	for _, t := range []struct{ name, slug string }{
		{"Go", "go"},
		{"Gin", "gin"},
		{"Vue3", "vue3"},
		{"随笔", "essay"},
	} {
		var count int64
		if err := db.Model(&Tag{}).Where("name = ?", t.name).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&Tag{Name: t.name, Slug: t.slug}).Error; err != nil {
			return nil, err
		}
	}
	return &category, nil
}

func seedPosts(db *gorm.DB, category *Category) error {
	var count int64
	if err := db.Model(&Post{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// tags by name
	findTag := func(name string) Tag {
		var tag Tag
		if err := db.Where("name = ?", name).First(&tag).Error; err != nil {
			logSeed("seedPosts: 未找到标签 " + name)
		}
		return tag
	}

	now := time.Now()
	type seedPost struct {
		post   Post
		tags   []string
		status int8
	}
	seeds := []seedPost{
		{post: Post{
			Title: "使用 Gin 搭建高性能博客后端", Slug: "gin-blog-backend", CategoryID: category.ID,
			Summary: "从零开始用 Go 语言与 Gin 框架搭建博客后端：路由分组、中间件链、GORM 数据访问与统一响应结构的最佳实践。",
			Content: seedArticleGin, Status: PostPublished,
			PublishedAt: ptrTime(now.AddDate(0, 0, -3)), CreatedAt: now.AddDate(0, 0, -3),
		}, tags: []string{"Go", "Gin"}, status: PostPublished},
		{post: Post{
			Title: "Vue3 组合式 API 在博客前台中的实践", Slug: "vue3-composition-api", CategoryID: category.ID,
			Summary: "组合式 API 改变了组织逻辑的方式：ref 与 reactive 管理状态，computed 派生数据，watch 处理副作用，让前台数据流清晰可测。",
			Content: seedArticleVue, Status: PostPublished,
			PublishedAt: ptrTime(now.AddDate(0, 0, -2)), CreatedAt: now.AddDate(0, 0, -2),
		}, tags: []string{"Vue3"}, status: PostPublished},
		{post: Post{
			Title: "Go 并发编程：goroutine 与 channel 入门", Slug: "go-concurrency", CategoryID: category.ID,
			Summary: "goroutine 轻量到可以同时运行数十万个，channel 让数据在协程之间安全流转。理解「不要通过共享内存来通信」是写出健壮并发代码的第一步。",
			Content: seedArticleConcurrency, Status: PostPublished,
			PublishedAt: ptrTime(now.AddDate(0, 0, -1)), CreatedAt: now.AddDate(0, 0, -1),
		}, tags: []string{"Go"}, status: PostPublished},
		{post: Post{
			Title: "博客改版计划（草稿）", Slug: "blog-redesign-draft", CategoryID: category.ID,
			Summary: "记录本次博客系统改版的目标、技术选型与待办事项，仍在撰写中，暂不发布。",
			Content: seedArticleDraft, Status: PostDraft, CreatedAt: now,
		}, tags: []string{"随笔"}, status: PostDraft},
	}

	for i := range seeds {
		s := &seeds[i]
		if err := db.Create(&s.post).Error; err != nil {
			return err
		}
		var tags []Tag
		for _, name := range s.tags {
			t := findTag(name)
			if t.ID != 0 {
				tags = append(tags, t)
			}
		}
		if len(tags) > 0 {
			if err := db.Model(&s.post).Association("Tags").Replace(tags); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedComments(db *gorm.DB) error {
	var count int64
	if err := db.Model(&Comment{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var post Post
	if err := db.Where("slug = ?", "gin-blog-backend").First(&post).Error; err != nil {
		return nil // 没有种子文章则跳过
	}

	zhangsan := Comment{
		PostID: post.ID, Nickname: "张三", Email: "zhangsan@example.com",
		Content: "写得很清楚，路由分组那一段正好解决了我一直没想明白的管理接口鉴权问题，感谢分享！",
		Status:  CommentApproved, IP: "192.168.1.20",
	}
	if err := db.Create(&zhangsan).Error; err != nil {
		return err
	}

	var admin User
	if err := db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		return err
	}
	reply := Comment{
		PostID: post.ID, ParentID: zhangsan.ID, Nickname: admin.Nickname, Email: admin.Email,
		Content: "感谢支持，后续会继续补充 GORM 关联查询与部署相关的内容，欢迎常来交流。",
		Status:  CommentApproved, IP: "127.0.0.1", IsAdmin: true,
	}
	if err := db.Create(&reply).Error; err != nil {
		return err
	}

	lisi := Comment{
		PostID: post.ID, Nickname: "李四", Email: "lisi@example.com",
		Content: "统一响应结构这个约定太实用了，前端一个拦截器就能处理所有错误，已经抄作业了。",
		Status:  CommentApproved, IP: "192.168.1.33",
	}
	return db.Create(&lisi).Error
}

func seedLinks(db *gorm.DB) {
	var count int64
	if err := db.Model(&Link{}).Count(&count).Error; err != nil || count > 0 {
		return
	}
	links := []Link{
		{Name: "Go 官方网站", URL: "https://go.dev", Description: "Go 语言官方网站与文档", Visible: true, Sort: 0},
		{Name: "Gin Web Framework", URL: "https://gin-gonic.com", Description: "Gin 框架官网", Visible: true, Sort: 1},
		{Name: "Vue.js 中文文档", URL: "https://cn.vuejs.org", Description: "Vue 3 官方中文文档", Visible: true, Sort: 2},
	}
	for i := range links {
		mustLog("seedLinks", db.Create(&links[i]).Error)
	}
}

func seedPage(db *gorm.DB) {
	var count int64
	if err := db.Model(&Page{}).Where("slug = ?", "about").Count(&count).Error; err != nil || count > 0 {
		return
	}
	page := Page{
		Title: "关于", Slug: "about", Status: 1,
		Content: "## 关于本站\n\n这里是我的个人博客，主要记录 Go 后端开发、前端工程化与日常随笔。\n\n本站使用 Go + Gin + GORM 构建后端，前台与管理端基于 Vue 3 + TypeScript，支持暗色模式、Markdown 渲染、代码高亮与 RSS 订阅。\n\n欢迎通过评论区交流，也可以通过友链页面找到我的常用站点。",
	}
	mustLog("seedPage", db.Create(&page).Error)
}

func logSeed(msg string) {
	println("[seed] " + msg)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

// ---------- 示例文章（Markdown，含二级标题与 Go/JS 代码块） ----------

const fence = "```"

const seedArticleGin = `## 为什么选择 Gin

Gin 是 Go 生态中最流行的 Web 框架之一，凭借基于基数树的路由匹配与极低的内存分配开销，它可以在单机上轻松支撑数万 QPS。对于一个个人博客系统来说，Gin 提供了恰到好处的抽象：分组路由、中间件链、参数绑定、JSON 序列化一应俱全，同时不会像大型全家桶框架那样带来沉重的学习成本与维护负担。

## 路由与中间件设计

博客后端的接口可以划分为公开接口与管理接口两组。公开接口挂载在 ` + fence + `/api/v1` + fence + ` 分组下，任何人都能访问；管理接口额外叠加 JWT 鉴权中间件，未携带合法令牌的请求会直接返回 401，由前端统一跳转到登录页。

` + fence + `go
func main() {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	api := r.Group("/api/v1")
	api.GET("/posts", handler.ListPosts)

	admin := api.Group("/admin", middleware.JWT())
	admin.GET("/posts", handler.AdminListPosts)

	_ = r.Run(":8080")
}
` + fence + `

中间件的执行顺序很重要：CORS 放在最外层保证预检请求不被拦截，Recovery 兜住所有 panic，JWT 只挂在管理分组上，公开接口完全无感。

## 数据访问

借助 GORM，我们可以用结构体描述表结构，用链式调用完成查询，AutoMigrate 让建表这件事彻底消失。配合纯 Go 的 SQLite 驱动，部署时免安装数据库服务，零配置即可启动，非常适合个人博客的场景。

统一响应结构 ` + fence + `{code, message, data}` + fence + ` 让前端可以用一个拦截器处理所有业务错误，这是前后端协作中最值得提前约定的事情之一。约定先行，联调时才能真正做到互不等待。`

const seedArticleVue = `## 从选项式到组合式

Vue 3 的组合式 API 把「一个功能」的所有逻辑收纳到一起，而不是按 data、methods、computed 人为切分。对于博客前台的文章列表、评论树、暗色模式切换这类状态联动较多的场景，组合式写法明显更清爽，也让逻辑可以被抽成可复用的组合式函数。

## 一个文章列表的组合式函数

` + fence + `js
import { ref, computed, onMounted } from 'vue'

export function usePosts() {
  const posts = ref([])
  const keyword = ref('')
  const filtered = computed(() =>
    posts.value.filter(p => p.title.includes(keyword.value))
  )

  onMounted(async () => {
    const res = await fetch('/api/v1/posts?pageSize=10')
    const body = await res.json()
    posts.value = body.data.list
  })

  return { keyword, filtered }
}
` + fence + `

把数据获取、派生状态、副作用封装在 ` + fence + `usePosts` + fence + ` 里，组件只负责渲染，测试时也可以脱离组件直接验证这段逻辑。

## 状态管理

Pinia 是 Vue 3 官方推荐的状态库。把登录态、站点设置放入 store 之后，任何组件都可以直接消费，配合持久化插件还能记住用户的暗色模式选择，刷新页面也不会丢失。

组合式 API 并不强制「推倒重来」，它更像一种更贴近 JavaScript 本身的表达方式：状态就是变量，副作用就在函数里，生命周期钩子只是注册时机的语法糖。理解这一点之后，再回头看选项式组件，会发现很多模板代码其实都可以消失。`

const seedArticleConcurrency = `## goroutine：轻量的执行单元

Go 的并发单元 goroutine 启动成本极低，初始栈只有几 KB，运行时会自动把它们多路复用到少量操作系统线程上。启动一个 goroutine 只需要一个 go 关键字，不需要手动管理线程池。

## channel：协程之间的桥梁

` + fence + `go
func worker(id int, jobs <-chan int, results chan<- int) {
	for j := range jobs {
		results <- j * j
	}
}

func main() {
	jobs := make(chan int, 100)
	results := make(chan int, 100)
	for w := 1; w <= 3; w++ {
		go worker(w, jobs, results)
	}
	for j := 1; j <= 9; j++ {
		jobs <- j
	}
	close(jobs)
	for a := 1; a <= 9; a++ {
		fmt.Println(<-results)
	}
}
` + fence + `

channel 分为有缓冲与无缓冲两种：无缓冲强调「交接完成」，有缓冲用于削峰。方向类型 ` + fence + `chan<-` + fence + ` 与 ` + fence + `<-chan` + fence + ` 能在编译期约束协程只能发送或只能接收，把并发协议写进类型系统。

## 并发安全的三条纪律

第一，所有 goroutine 都要有明确的退出路径，谁启动谁负责停止，避免泄漏；第二，共享状态要么用 channel 传递，要么用锁保护，永远不要裸读写；第三，用 ` + fence + `go test -race` + fence + ` 把数据竞争消灭在上线之前。

记住 Rob Pike 的那句名言：不要通过共享内存来通信，而要通过通信来共享内存。理解了这句话，Go 并发编程就入门了。`

const seedArticleDraft = `## 改版目标

记录本次博客系统改版的目标：前后端分离、接口契约先行、支持暗色模式与移动端响应式、评论免注册但需审核。

## 待办

- 补充部署方案（Docker Compose + MySQL）
- 整理文章写作模板
- 优化评论审核体验`
