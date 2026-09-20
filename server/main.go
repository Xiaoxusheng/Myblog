// MyBlog 后端服务：Gin + GORM，默认 SQLite 零配置启动，支持 MySQL 部署。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myblog/server/config"
	"myblog/server/handler"
	"myblog/server/model"
	"myblog/server/router"
)

// version 构建时可用 -ldflags "-X main.version=v1.2.0" 注入；默认 dev
var version = "dev"

func main() {
	cfg := config.Load()
	handler.SetVersion(version)

	db, err := model.Open(cfg)
	if err != nil {
		log.Fatalf("启动失败：连接数据库（BLOG_DB_TYPE=%s）：%v", cfg.DBType, err)
	}
	if err := model.AutoMigrate(db); err != nil {
		log.Fatalf("启动失败：迁移表结构：%v", err)
	}
	if err := model.Seed(db); err != nil {
		log.Fatalf("启动失败：初始化种子数据：%v", err)
	}
	// 存量明文敏感字段（评论邮箱/IP、管理员邮箱）一次性加密，幂等
	if err := model.MigratePIIEncryption(db); err != nil {
		log.Fatalf("启动失败：加密存量敏感字段：%v", err)
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("启动失败：创建上传目录 %s：%v", cfg.UploadDir, err)
	}

	// 定时发布调度：启动时先补跑一次（接管停机期间到期的计划），再按间隔轮询。
	// 计划存在数据库中，重启不丢；到点文章/页面 status 3→1（详见 model.PublishDueScheduledPosts）。
	publishDue := func(when string) {
		n, err := model.PublishDueScheduledPosts(db)
		if err != nil {
			log.Printf("scheduler: %s扫描到期定时文章失败：%v", when, err)
		} else if n > 0 {
			log.Printf("scheduler: %s已发布 %d 篇到期定时文章", when, n)
			// 通知中心：定时文章已上线（契约 #73 type=post_published）
			_ = model.CreateNotification(db, "post_published", "定时发布完成",
				fmt.Sprintf("%d 篇到期的定时文章已自动发布", n), "/posts?status=1")
		}

		// 页面与文章共用同一调度周期与状态语义
		pn, perr := model.PublishDueScheduledPages(db)
		if perr != nil {
			log.Printf("scheduler: %s扫描到期定时页面失败：%v", when, perr)
		} else if pn > 0 {
			log.Printf("scheduler: %s已发布 %d 个到期定时页面", when, pn)
			_ = model.CreateNotification(db, "post_published", "定时发布完成",
				fmt.Sprintf("%d 个到期的定时页面已自动发布", pn), "/pages?status=1")
		}
	}
	publishDue("启动补跑：")
	go func() {
		ticker := time.NewTicker(cfg.ScheduleInterval)
		defer ticker.Stop()
		for range ticker.C {
			publishDue("定时扫描：")
		}
	}()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.Setup(cfg),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("MyBlog server 已启动，监听 :%s（DB=%s）", cfg.Port, cfg.DBType)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP 服务异常退出：%v", err)
		}
	}()

	// 优雅关闭：等待中断信号，超时后强制退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("收到退出信号，正在优雅关闭……")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("优雅关闭失败：%v", err)
	}
	log.Println("MyBlog server 已退出")
}
