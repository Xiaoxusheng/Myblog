// Package router 装配全部路由：/api/v1/*、/uploads/*、/rss
package router

import (
	"myblog/server/config"
	"myblog/server/handler"
	"myblog/server/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 构建并返回 gin 引擎
func Setup(cfg *config.Config) *gin.Engine {
	handler.SetConfig(cfg)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())
	r.MaxMultipartMemory = 12 << 20

	// 上传文件静态服务
	r.Static("/uploads", cfg.UploadDir)

	// RSS / Sitemap / Robots（根路径特例）
	r.GET("/rss", handler.RSS)
	r.GET("/sitemap.xml", handler.Sitemap)
	r.GET("/robots.txt", handler.Robots)

	api := r.Group("/api/v1")
	{
		// ---------- 公开接口（无需登录） ----------
		api.GET("/site", handler.GetSite)
		api.GET("/posts", handler.ListPosts)
		api.GET("/posts/:slug", handler.GetPost)
		api.GET("/posts/:slug/comments", handler.ListComments)
		api.POST("/posts/:slug/comments", middleware.CommentRateLimit(), handler.CreateComment)
		api.POST("/posts/:slug/like", handler.LikePost)
		api.GET("/archive", handler.Archive)
		api.GET("/pages/:slug", handler.GetPage)
		api.GET("/links", handler.ListLinks)
	}

	// 登录不走 JWT
	adminLogin := api.Group("/admin")
	{
		adminLogin.POST("/auth/login", handler.Login)
	}

	// ---------- 管理接口（Bearer JWT） ----------
	admin := api.Group("/admin", middleware.JWT([]byte(cfg.JWTSecret)))
	{
		// 认证
		admin.GET("/auth/me", handler.Me)
		admin.PUT("/auth/password", handler.UpdatePassword)
		admin.PUT("/auth/profile", handler.UpdateProfile)

		// 仪表盘
		admin.GET("/stats", handler.Stats)

		// 文章
		admin.GET("/posts", handler.AdminListPosts)
		admin.POST("/posts", handler.AdminCreatePost)
		admin.GET("/posts/:id", handler.AdminGetPost)
		admin.PUT("/posts/:id", handler.AdminUpdatePost)
		admin.PUT("/posts/:id/status", handler.AdminUpdatePostStatus)
		admin.DELETE("/posts/:id", handler.AdminDeletePost)

		// 分类
		admin.GET("/categories", handler.AdminListCategories)
		admin.POST("/categories", handler.AdminCreateCategory)
		admin.PUT("/categories/:id", handler.AdminUpdateCategory)
		admin.DELETE("/categories/:id", handler.AdminDeleteCategory)

		// 标签
		admin.GET("/tags", handler.AdminListTags)
		admin.POST("/tags", handler.AdminCreateTag)
		admin.PUT("/tags/:id", handler.AdminUpdateTag)
		admin.DELETE("/tags/:id", handler.AdminDeleteTag)

		// 评论
		admin.GET("/comments", handler.AdminListComments)
		admin.PUT("/comments/:id/status", handler.AdminUpdateCommentStatus)
		admin.POST("/comments/:id/reply", handler.AdminReplyComment)
		admin.DELETE("/comments/:id", handler.AdminDeleteComment)

		// 友链
		admin.GET("/links", handler.AdminListLinks)
		admin.POST("/links", handler.AdminCreateLink)
		admin.PUT("/links/:id", handler.AdminUpdateLink)
		admin.DELETE("/links/:id", handler.AdminDeleteLink)

		// 页面
		admin.GET("/pages", handler.AdminListPages)
		admin.POST("/pages", handler.AdminCreatePage)
		admin.GET("/pages/:id", handler.AdminGetPage)
		admin.PUT("/pages/:id", handler.AdminUpdatePage)
		admin.DELETE("/pages/:id", handler.AdminDeletePage)

		// 设置
		admin.GET("/settings", handler.AdminGetSettings)
		admin.PUT("/settings", handler.AdminUpdateSettings)

		// 媒体
		admin.POST("/uploads", handler.AdminUpload)
		admin.GET("/uploads", handler.AdminListUploads)
		admin.DELETE("/uploads/:id", handler.AdminDeleteUpload)
	}

	return r
}
