package router

import (
	"log"
	"net/http"
	"os"
	"opsmanage/internal/config"
	"opsmanage/internal/handler"
	"opsmanage/internal/middleware"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewServer(cfg *config.Config) *http.Server {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Login rate limiter: 10 requests per minute per IP
	loginLimiter := middleware.NewRateLimiter(10, time.Minute)

	r := gin.Default()

	r.Use(middleware.CORS())
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityCheck())

	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// Swagger API 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", loginLimiter.Middleware(), handler.Login)
			auth.POST("/logout", handler.Logout)
			auth.GET("/captcha", handler.Captcha)
		}

		secure := api.Group("")
		secure.Use(middleware.JWTAuth())
		{
			secure.GET("/profile", handler.GetProfile)
			secure.PUT("/password", handler.ChangePassword)

			// 用户管理（仅管理员）
			userMgmt := secure.Group("")
			userMgmt.Use(middleware.AdminOnly())
			{
				userMgmt.POST("/auth/register", handler.Register)
				userMgmt.GET("/users", handler.ListUsers)
				userMgmt.DELETE("/users/:id", handler.DeleteUser)
			}

			dashboard := secure.Group("/dashboard")
			{
				dashboard.GET("", handler.GetDashboard)
				dashboard.GET("/system-info", handler.GetSystemInfo)
				dashboard.GET("/system-status", handler.GetSystemStatus)
			}

			site := secure.Group("/websites")
			{
				site.GET("", handler.ListWebsites)
				site.POST("", handler.CreateWebsite)
				site.GET("/:id", handler.GetWebsite)
				site.PUT("/:id", handler.UpdateWebsite)
				site.DELETE("/:id", handler.DeleteWebsite)
				site.POST("/:id/start", handler.StartWebsite)
				site.POST("/:id/stop", handler.StopWebsite)
			}

		db := secure.Group("/databases")
		{
			db.GET("", handler.ListDatabases)
			db.POST("", handler.CreateDatabase)
			db.GET("/:id", handler.GetDatabase)
			db.PUT("/:id", handler.UpdateDatabase)
			db.DELETE("/:id", handler.DeleteDatabase)
			db.POST("/:id/start", handler.StartDatabase)
			db.POST("/:id/stop", handler.StopDatabase)

			// 数据库服务管理（新）
			db.GET("/services/status", handler.DBServiceStatus)
			db.GET("/instances", handler.ListDBInstances)
			db.POST("/instances", handler.CreateDBInstance)
			db.GET("/instances/:id", handler.GetDBInstance)
			db.POST("/instances/:id/:action", handler.DBInstanceAction)
			db.GET("/instances/:id/config", handler.DBInstanceConfig)
			db.PUT("/instances/:id/config", handler.DBInstanceConfig)
			db.GET("/instances/:id/stats", handler.DBInstanceStats)

			// 数据库管理
			db.GET("/instances/:id/databases", handler.ListDBDatabases)
			db.POST("/instances/:id/databases", handler.CreateDBDatabase)
			db.DELETE("/databases/:did", handler.DeleteDBDatabase)
			db.POST("/instances/:id/databases/sync", handler.SyncDBDatabases)

			// 用户管理
			db.GET("/instances/:id/users", handler.ListDBUsers)
			db.POST("/instances/:id/users", handler.CreateDBUser)
			db.DELETE("/users/:did", handler.DeleteDBUser)

			// 备份管理
			db.GET("/instances/:id/backups", handler.ListDBBackups)
			db.POST("/instances/:id/backups", handler.CreateDBBackup)
			db.POST("/backups/:bid/restore", handler.RestoreDBBackup)
		}

		// Nginx 管理
		nginx := secure.Group("/nginx")
		{
			nginx.GET("/status", handler.NginxStatus)
			nginx.GET("/overview", handler.NginxStatusOverview)
			nginx.POST("/install", handler.NginxInstall)
			nginx.POST("/service", handler.NginxService)
			nginx.GET("/test", handler.NginxTestConfig)
			nginx.POST("/import", handler.NginxImportSites)

			// 站点管理
			nginx.GET("/sites", handler.ListNginxSites)
			nginx.POST("/sites", handler.CreateNginxSite)
			nginx.GET("/sites/:id", handler.GetNginxSite)
			nginx.PUT("/sites/:id", handler.UpdateNginxSite)
			nginx.DELETE("/sites/:id", handler.DeleteNginxSite)
			nginx.POST("/sites/:id/:action", handler.NginxSiteAction)
			nginx.POST("/sites/:id/reload", handler.NginxSiteReload)

			// SSL 管理
			nginx.POST("/sites/:id/ssl", handler.NginxSiteSSL)

			// 配置编辑
			nginx.GET("/sites/:id/config", handler.NginxSiteConfig)
			nginx.PUT("/sites/:id/config", handler.NginxSiteConfig)

			// 日志查看
			nginx.GET("/sites/:id/logs", handler.NginxSiteLogs)
			nginx.GET("/sites/:id/logs/ws", handler.NginxSiteLogStream)
		}

			container := secure.Group("/containers")
			{
				container.GET("", handler.ListContainers)
				container.POST("", handler.CreateContainer)
				container.GET("/:id", handler.GetContainer)
				container.DELETE("/:id", handler.DeleteContainer)
				container.POST("/:id/start", handler.StartContainer)
				container.POST("/:id/stop", handler.StopContainer)
				container.POST("/:id/restart", handler.RestartContainer)
				container.GET("/:id/logs", handler.GetContainerLogs)
			container.GET("/images", handler.ListImages)
			container.POST("/images/pull", handler.PullImage)
			container.GET("/overview", handler.GetDockerOverview)
			container.GET("/networks", handler.ListDockerNetworks)
			container.DELETE("/networks/:id", handler.RemoveNetwork)
			container.GET("/volumes", handler.ListDockerVolumes)
			container.DELETE("/volumes/:id", handler.RemoveVolume)
			container.DELETE("/images/:id", handler.RemoveImage)
			container.POST("/prune", handler.PruneDocker)
			container.GET("/registries", handler.ListRegistries)
			container.POST("/registries", handler.CreateRegistry)
			container.DELETE("/registries/:id", handler.DeleteRegistry)
			container.GET("/compose", handler.ListComposeProjects)
			container.POST("/compose", handler.CreateComposeProject)
			container.DELETE("/compose/:id", handler.DeleteComposeProject)
			container.POST("/compose/:id/start", handler.StartComposeProject)
			container.POST("/compose/:id/stop", handler.StopComposeProject)
			container.GET("/templates", handler.ListComposeTemplates)
			container.POST("/templates", handler.CreateComposeTemplate)
			container.DELETE("/templates/:id", handler.DeleteComposeTemplate)
		}

			files := secure.Group("/files")
			{
				files.GET("/list", handler.ListFiles)
				files.GET("/read", handler.ReadFile)
				files.POST("/save", handler.SaveFile)
				files.GET("/download", handler.DownloadFile)
				files.POST("/upload", handler.UploadFile)
				files.POST("/rename", handler.RenameFile)
				files.DELETE("", handler.DeleteFile)
				files.POST("/mkdir", handler.Mkdir)
				files.POST("/copy", handler.CopyFile)
			}

		security := secure.Group("/security")
		{
			security.GET("/rules", handler.ListSecurityRules)
			security.POST("/rules", handler.CreateSecurityRule)
			security.GET("/rules/:id", handler.GetSecurityRule)
			security.PUT("/rules/:id", handler.UpdateSecurityRule)
			security.DELETE("/rules/:id", handler.DeleteSecurityRule)
			security.POST("/rules/:id/toggle", handler.ToggleSecurityRule)

			// SSH 管理
			security.GET("/ssh", handler.ListSSHAccounts)
			security.POST("/ssh", handler.CreateSSHAccount)
			security.GET("/ssh/:id", handler.GetSSHAccount)
			security.GET("/ssh/:id/full", handler.GetSSHAccountFull)
			security.PUT("/ssh/:id", handler.UpdateSSHAccount)
			security.DELETE("/ssh/:id", handler.DeleteSSHAccount)
			security.POST("/ssh/:id/test", handler.TestSSHConnection)
			security.POST("/ssh/:id/credential", handler.ChangeSSHCredential)
			security.POST("/ssh/:id/change-password", handler.ChangeRemotePassword)
			security.POST("/ssh/:id/change-port", handler.ChangeSSHPort)
			security.POST("/ssh/:id/restart", handler.RestartSSHD)
			security.POST("/ssh/:id/install-key", handler.InstallSSHKey)
			security.POST("/ssh/:id/command", handler.ExecuteSSHCommand)
			security.GET("/ssh/:id/sshd-config", handler.GetSSHdConfig)
			security.PUT("/ssh/:id/sshd-config", handler.SaveSSHdConfig)

			// 防火墙管理
			security.GET("/firewall/status", handler.GetFirewallStatus)
			security.POST("/firewall/rules", handler.AddFirewallRule)
			security.DELETE("/firewall/rules/:id", handler.DeleteFirewallRule)
			security.GET("/firewall/ports", handler.GetFirewallPorts)
			security.POST("/firewall/restart", handler.RestartFirewall)
		}

		tasks := secure.Group("/tasks")
		{
			tasks.GET("", handler.ListTasks)
			tasks.POST("", handler.CreateTask)
			tasks.GET("/:id", handler.GetTask)
			tasks.PUT("/:id", handler.UpdateTask)
			tasks.DELETE("/:id", handler.DeleteTask)
			tasks.POST("/:id/run", handler.RunTask)
			tasks.POST("/:id/toggle", handler.ToggleTask)
		}

		logs := secure.Group("/logs")
		{
			logs.GET("", handler.ListLogs)
			logs.GET("/sources", handler.GetLogSources)
			logs.DELETE("/clear", handler.ClearLogs)
			logs.GET("/system", handler.GetSystemLogs)
			logs.GET("/ssh", handler.GetSSHLogs)
		}

		settings := secure.Group("/settings")
		{
			settings.GET("", handler.GetSettings)
			settings.PUT("", handler.UpdateSettings)
			settings.GET("/:key", handler.GetSettingByKey)
		}
		}
	}

	r.GET("/ws", handler.WSFileHandler)
	r.GET("/ws/ssh/:id", handler.WebSSHHandler)

	r.NoRoute(func(c *gin.Context) {
		if _, err := os.Stat("./static" + c.Request.URL.Path); err == nil {
			c.File("./static" + c.Request.URL.Path)
		} else {
			c.File("./static/index.html")
		}
	})

	addr := cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	if cfg.Server.TLSCert != "" && cfg.Server.TLSKey != "" {
		log.Printf("HTTPS 模式启用, 证书: %s", cfg.Server.TLSCert)
		srv.TLSConfig = nil // Gin handles TLS via ListenAndServeTLS
		// We'll call srv.ListenAndServeTLS in main
	}
	return srv
}
