package main

import (
	"log"
	"opsmanage/internal/config"
	"opsmanage/internal/middleware"
	"opsmanage/internal/router"
	"opsmanage/internal/scheduler"
)

func main() {
	cfg := config.Load()

	if err := config.InitDB(cfg); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	config.InitStatic()

	scheduler.Init()
	middleware.StartBlacklistCleanup()

	log.Printf("🚀 OpsManage 启动中，监听 %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := router.Run(cfg); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
