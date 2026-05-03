package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opsmanage/internal/config"
	"opsmanage/internal/middleware"
	"opsmanage/internal/router"
	"opsmanage/internal/scheduler"
)

// @title           OpsManage API
// @version         1.0.0
// @description     轻量级服务器运维管理面板 API，提供网站管理、数据库管理、Docker 容器管理、文件管理、计划任务、安全管理等功能。
// @host            localhost:9090
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     JWT Bearer Token，格式: Bearer {token}

func main() {
	cfg := config.Load()

	if err := config.InitDB(cfg); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	config.InitStatic()

	scheduler.Init()
	middleware.StartBlacklistCleanup()

	srv := router.NewServer(cfg)

	go func() {
		log.Printf("🚀 OpsManage 启动中，监听 %s", srv.Addr)
		var err error
		if cfg.Server.TLSCert != "" && cfg.Server.TLSKey != "" {
			err = srv.ListenAndServeTLS(cfg.Server.TLSCert, cfg.Server.TLSKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务关闭失败: %v", err)
	}
	log.Println("服务已安全退出")
}
