// MyBlog 后端服务：Gin + GORM，默认 SQLite 零配置启动，支持 MySQL 部署。
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myblog/server/config"
	"myblog/server/model"
	"myblog/server/router"
)

func main() {
	cfg := config.Load()

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
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("启动失败：创建上传目录 %s：%v", cfg.UploadDir, err)
	}

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
