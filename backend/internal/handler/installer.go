package handler

import (
	"fmt"
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ========== 进度追踪系统 ==========

type TaskProgress struct {
	TaskID    string    `json:"task_id"`
	Type      string    `json:"type"`      // nginx, mysql, postgresql, redis
	Status    string    `json:"status"`    // pending, running, success, failed
	Progress  int       `json:"progress"`  // 0-100
	Message   string    `json:"message"`
	Detail    string    `json:"detail"`
	StartTime time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

var (
	progressStore = make(map[string]*TaskProgress)
	progressMutex sync.RWMutex
)

func setProgress(taskID, status string, progress int, message, detail string) {
	progressMutex.Lock()
	defer progressMutex.Unlock()
	
	if p, ok := progressStore[taskID]; ok {
		p.Status = status
		p.Progress = progress
		p.Message = message
		p.Detail = detail
		if status == "success" || status == "failed" {
			now := time.Now()
			p.EndTime = &now
		}
	}
}

func createProgress(taskID, taskType string) *TaskProgress {
	progressMutex.Lock()
	defer progressMutex.Unlock()
	
	p := &TaskProgress{
		TaskID:    taskID,
		Type:      taskType,
		Status:    "pending",
		Progress:  0,
		Message:   "准备中...",
		StartTime: time.Now(),
	}
	progressStore[taskID] = p
	return p
}

func getProgress(taskID string) *TaskProgress {
	progressMutex.RLock()
	defer progressMutex.RUnlock()
	return progressStore[taskID]
}

// GetTaskProgress 获取任务进度
func GetTaskProgress(c *gin.Context) {
	taskID := c.Param("taskId")
	p := getProgress(taskID)
	if p == nil {
		fail(c, 404, "任务不存在")
		return
	}
	success(c, p)
}

// GetActiveTasks 获取所有活跃任务
func GetActiveTasks(c *gin.Context) {
	progressMutex.RLock()
	defer progressMutex.RUnlock()
	
	tasks := []TaskProgress{}
	for _, p := range progressStore {
		if p.Status == "running" || p.Status == "pending" {
			tasks = append(tasks, *p)
		}
	}
	success(c, tasks)
}

// StreamTaskProgress SSE 实时推送进度
func StreamTaskProgress(c *gin.Context) {
	taskID := c.Param("taskId")
	
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	
	flusher, ok := c.Writer.(gin.ResponseWriter)
	if !ok {
		c.JSON(500, gin.H{"error": "SSE not supported"})
		return
	}
	
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	lastStatus := ""
	
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			p := getProgress(taskID)
			if p == nil {
				fmt.Fprintf(c.Writer, "data: {\"error\":\"task not found\"}\n\n")
				flusher.Flush()
				return
			}
			
			// 只在状态变化时发送
			currentStatus := fmt.Sprintf("%s-%d-%s", p.Status, p.Progress, p.Message)
			if currentStatus != lastStatus {
				data := fmt.Sprintf("{\"task_id\":\"%s\",\"type\":\"%s\",\"status\":\"%s\",\"progress\":%d,\"message\":\"%s\",\"detail\":\"%s\"}\n",
					p.TaskID, p.Type, p.Status, p.Progress, p.Message, p.Detail)
				fmt.Fprintf(c.Writer, "data: %s\n\n", data)
				flusher.Flush()
				lastStatus = currentStatus
			}
			
			// 任务完成或失败时关闭连接
			if p.Status == "success" || p.Status == "failed" {
				return
			}
		}
	}
}

// ========== Nginx 安装 ==========

// NginxInstall 安装 Nginx
func NginxInstall(c *gin.Context) {
	if isNginxInstalled() {
		fail(c, 400, "Nginx 已安装")
		return
	}
	
	taskID := "nginx-install-" + time.Now().Format("20060102150405")
	createProgress(taskID, "nginx")
	
	go func() {
		setProgress(taskID, "running", 10, "正在更新软件源...", "")
		addLog("info", "nginx", "开始安装 Nginx...")
		
		// 更新软件源
		cmd := exec.Command("apt-get", "update", "-qq")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		output, err := cmd.CombinedOutput()
		if err != nil {
			setProgress(taskID, "failed", 10, "更新软件源失败", string(output))
			addLog("error", "nginx", "更新软件源失败: "+string(output))
			return
		}
		
		setProgress(taskID, "running", 40, "正在安装 Nginx...", "")
		
		// 安装 nginx
		cmd = exec.Command("apt-get", "install", "-y", "-qq", "nginx")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		output, err = cmd.CombinedOutput()
		if err != nil {
			setProgress(taskID, "failed", 40, "安装 Nginx 失败", string(output))
			addLog("error", "nginx", "安装 Nginx 失败: "+string(output))
			return
		}
		
		setProgress(taskID, "running", 70, "正在配置服务...", "")
		
		// 启用并启动 nginx
		exec.Command("systemctl", "enable", "nginx").Run()
		exec.Command("systemctl", "start", "nginx").Run()
		
		setProgress(taskID, "running", 90, "正在验证安装...", "")
		time.Sleep(1 * time.Second)
		
		if isNginxRunning() {
			setProgress(taskID, "success", 100, "Nginx 安装成功！", "版本: "+getNginxVersion())
			addLog("info", "nginx", "Nginx 安装成功")
		} else {
			setProgress(taskID, "failed", 90, "Nginx 安装完成但未运行", "请检查 systemctl status nginx")
			addLog("warning", "nginx", "Nginx 安装完成但未运行")
		}
	}()
	
	success(c, gin.H{"task_id": taskID, "message": "正在安装 Nginx..."})
}

// ========== MySQL 安装 ==========

// MySQLInstall 安装 MySQL
func MySQLInstall(c *gin.Context) {
	if isMySQLInstalled() {
		fail(c, 400, "MySQL 已安装")
		return
	}
	
	var req struct {
		RootPass string `json:"root_pass"`
	}
	c.ShouldBindJSON(&req)
	
	taskID := "mysql-install-" + time.Now().Format("20060102150405")
	createProgress(taskID, "mysql")
	
	go func() {
		setProgress(taskID, "running", 5, "正在准备安装环境...", "")
		addLog("info", "database", "开始安装 MySQL...")
		
		// 清理旧的 MySQL 仓库问题
		setProgress(taskID, "running", 10, "正在清理旧仓库配置...", "")
		exec.Command("rm", "-f", "/etc/apt/sources.list.d/mysql.list").Run()
		
		// 更新软件源
		setProgress(taskID, "running", 20, "正在更新软件源...", "")
		cmd := exec.Command("apt-get", "update", "-qq")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		output, err := cmd.CombinedOutput()
		if err != nil {
			setProgress(taskID, "failed", 20, "更新软件源失败", string(output))
			addLog("error", "database", "更新软件源失败: "+string(output))
			return
		}
		
		setProgress(taskID, "running", 40, "正在安装 MySQL Server...", "这可能需要几分钟")
		
		// 安装 mysql-server
		cmd = exec.Command("apt-get", "install", "-y", "-qq", "mysql-server")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		output, err = cmd.CombinedOutput()
		if err != nil {
			setProgress(taskID, "failed", 40, "安装 MySQL 失败", string(output))
			addLog("error", "database", "安装 MySQL 失败: "+string(output))
			return
		}
		
		setProgress(taskID, "running", 70, "正在配置 MySQL 服务...", "")
		
		// 启用并启动 mysql
		exec.Command("systemctl", "enable", "mysql").Run()
		exec.Command("systemctl", "start", "mysql").Run()
		
		setProgress(taskID, "running", 80, "正在配置安全设置...", "")
		
		// 设置 root 密码
		if req.RootPass != "" {
			cmd = exec.Command("mysql", "-e", fmt.Sprintf("ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '%s'; FLUSH PRIVILEGES;", req.RootPass))
			cmd.Run()
		}
		
		setProgress(taskID, "running", 95, "正在验证安装...", "")
		time.Sleep(2 * time.Second)
		
		if isMySQLRunning() {
			version := getMySQLVersion()
			setProgress(taskID, "success", 100, "MySQL 安装成功！", "版本: "+version)
			addLog("info", "database", "MySQL 安装成功, 版本: "+version)
			
			// 更新数据库实例记录
			config.DB.Model(&model.DBInstance{}).Where("type = ?", "mysql").Updates(map[string]interface{}{
				"status":  "running",
				"version": version,
			})
		} else {
			setProgress(taskID, "failed", 95, "MySQL 安装完成但未运行", "请检查 systemctl status mysql")
			addLog("warning", "database", "MySQL 安装完成但未运行")
		}
	}()
	
	success(c, gin.H{"task_id": taskID, "message": "正在安装 MySQL..."})
}

// ========== PostgreSQL 安装 ==========

// PostgreSQLInstall 安装 PostgreSQL
func PostgreSQLInstall(c *gin.Context) {
	if isPostgreSQLInstalled() {
		fail(c, 400, "PostgreSQL 已安装")
		return
	}
	
	taskID := "pg-install-" + time.Now().Format("20060102150405")
	createProgress(taskID, "postgresql")
	
	go func() {
		setProgress(taskID, "running", 10, "正在更新软件源...", "")
		addLog("info", "database", "开始安装 PostgreSQL...")
		
		cmd := exec.Command("apt-get", "update", "-qq")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		output, err := cmd.CombinedOutput()
		if err != nil {
			setProgress(taskID, "failed", 10, "更新软件源失败", string(output))
			return
		}
		
		setProgress(taskID, "running", 40, "正在安装 PostgreSQL...", "这可能需要几分钟")
		
		cmd = exec.Command("apt-get", "install", "-y", "-qq", "postgresql", "postgresql-contrib")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		output, err = cmd.CombinedOutput()
		if err != nil {
			setProgress(taskID, "failed", 40, "安装 PostgreSQL 失败", string(output))
			addLog("error", "database", "安装 PostgreSQL 失败: "+string(output))
			return
		}
		
		setProgress(taskID, "running", 80, "正在配置服务...", "")
		exec.Command("systemctl", "enable", "postgresql").Run()
		exec.Command("systemctl", "start", "postgresql").Run()
		
		setProgress(taskID, "running", 95, "正在验证安装...", "")
		time.Sleep(2 * time.Second)
		
		if isPostgreSQLRunning() {
			version := getPostgreSQLVersion()
			setProgress(taskID, "success", 100, "PostgreSQL 安装成功！", "版本: "+version)
			addLog("info", "database", "PostgreSQL 安装成功, 版本: "+version)
			
			config.DB.Model(&model.DBInstance{}).Where("type = ?", "postgresql").Updates(map[string]interface{}{
				"status":  "running",
				"version": version,
			})
		} else {
			setProgress(taskID, "failed", 95, "PostgreSQL 安装完成但未运行", "请检查 systemctl status postgresql")
		}
	}()
	
	success(c, gin.H{"task_id": taskID, "message": "正在安装 PostgreSQL..."})
}

// ========== Redis 安装 ==========

// RedisInstall 安装 Redis
func RedisInstall(c *gin.Context) {
	if isRedisInstalled() {
		fail(c, 400, "Redis 已安装")
		return
	}
	
	taskID := "redis-install-" + time.Now().Format("20060102150405")
	createProgress(taskID, "redis")
	
	go func() {
		setProgress(taskID, "running", 10, "正在更新软件源...", "")
		addLog("info", "database", "开始安装 Redis...")
		
		cmd := exec.Command("apt-get", "update", "-qq")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		output, err := cmd.CombinedOutput()
		if err != nil {
			setProgress(taskID, "failed", 10, "更新软件源失败", string(output))
			return
		}
		
		setProgress(taskID, "running", 50, "正在安装 Redis...", "")
		
		cmd = exec.Command("apt-get", "install", "-y", "-qq", "redis-server")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		output, err = cmd.CombinedOutput()
		if err != nil {
			setProgress(taskID, "failed", 50, "安装 Redis 失败", string(output))
			addLog("error", "database", "安装 Redis 失败: "+string(output))
			return
		}
		
		setProgress(taskID, "running", 80, "正在配置服务...", "")
		exec.Command("systemctl", "enable", "redis-server").Run()
		exec.Command("systemctl", "start", "redis-server").Run()
		
		setProgress(taskID, "running", 95, "正在验证安装...", "")
		time.Sleep(1 * time.Second)
		
		if isRedisRunning() {
			version := getRedisVersion()
			setProgress(taskID, "success", 100, "Redis 安装成功！", "版本: "+version)
			addLog("info", "database", "Redis 安装成功, 版本: "+version)
			
			config.DB.Model(&model.DBInstance{}).Where("type = ?", "redis").Updates(map[string]interface{}{
				"status":  "running",
				"version": version,
			})
		} else {
			setProgress(taskID, "failed", 95, "Redis 安装完成但未运行", "请检查 systemctl status redis-server")
		}
	}()
	
	success(c, gin.H{"task_id": taskID, "message": "正在安装 Redis..."})
}

// ========== 通用安装接口 ==========

// InstallService 统一安装入口
func InstallService(c *gin.Context) {
	serviceType := c.Param("type")
	
	switch strings.ToLower(serviceType) {
	case "nginx":
		NginxInstall(c)
	case "mysql":
		MySQLInstall(c)
	case "postgresql", "postgres":
		PostgreSQLInstall(c)
	case "redis":
		RedisInstall(c)
	default:
		fail(c, 400, "不支持的服务类型: "+serviceType)
	}
}
