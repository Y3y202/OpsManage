package handler

import (
	"bufio"
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
	TaskID    string     `json:"task_id"`
	Type      string     `json:"type"`
	Status    string     `json:"status"`
	Progress  int        `json:"progress"`
	Message   string     `json:"message"`
	Logs      []string   `json:"logs"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

var (
	progressStore = make(map[string]*TaskProgress)
	progressMutex sync.RWMutex
)

func setProgress(taskID, status string, progress int, message string) {
	progressMutex.Lock()
	defer progressMutex.Unlock()

	if p, ok := progressStore[taskID]; ok {
		p.Status = status
		p.Progress = progress
		p.Message = message
		if status == "success" || status == "failed" {
			now := time.Now()
			p.EndTime = &now
		}
	}
}

func addInstallLog(taskID, log string) {
	progressMutex.Lock()
	defer progressMutex.Unlock()

	if p, ok := progressStore[taskID]; ok {
		p.Logs = append(p.Logs, log)
		if len(p.Logs) > 100 {
			p.Logs = p.Logs[len(p.Logs)-100:]
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
		Logs:      []string{},
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

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	lastLogCount := 0
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

			progressMutex.RLock()
			currentStatus := fmt.Sprintf("%s-%d-%s-%d", p.Status, p.Progress, p.Message, len(p.Logs))
			logs := make([]string, len(p.Logs))
			copy(logs, p.Logs)
			progressMutex.RUnlock()

			// 状态变化或有新日志时发送
			if currentStatus != lastStatus {
				// 发送新日志
				newLogs := logs[lastLogCount:]
				for _, log := range newLogs {
					logData := fmt.Sprintf("{\"type\":\"log\",\"content\":\"%s\"}\n", escapeJSON(log))
					fmt.Fprintf(c.Writer, "data: %s\n\n", logData)
					flusher.Flush()
				}
				lastLogCount = len(logs)

				// 发送进度更新
				data := fmt.Sprintf("{\"type\":\"progress\",\"task_id\":\"%s\",\"service\":\"%s\",\"status\":\"%s\",\"progress\":%d,\"message\":\"%s\",\"log_count\":%d}\n",
					p.TaskID, p.Type, p.Status, p.Progress, escapeJSON(p.Message), len(p.Logs))
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

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// ========== 安装脚本执行 ==========

// runInstallScript 执行安装脚本
func runInstallScript(taskID, service string, args ...string) {
	scriptPath := "./scripts/install.sh"

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = "/item/OpsManage/backend/scripts/install.sh"
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			setProgress(taskID, "failed", 0, "安装脚本不存在")
			return
		}
	}

	os.Chmod(scriptPath, 0755)

	cmdArgs := []string{service}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(scriptPath, cmdArgs...)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		setProgress(taskID, "failed", 0, "创建管道失败: "+err.Error())
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		setProgress(taskID, "failed", 0, "启动安装脚本失败: "+err.Error())
		return
	}

	setProgress(taskID, "running", 5, "开始安装...")

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "PROGRESS:") {
			parts := strings.SplitN(line[9:], ":", 2)
			if len(parts) == 2 {
				progress := 0
				fmt.Sscanf(parts[0], "%d", &progress)
				setProgress(taskID, "running", progress, parts[1])
			}
		} else if strings.HasPrefix(line, "INFO:") {
			addInstallLog(taskID, "ℹ️ "+line[5:])
		} else if strings.HasPrefix(line, "ERROR:") {
			addInstallLog(taskID, "❌ "+line[6:])
		} else if strings.HasPrefix(line, "SUCCESS:") {
			addInstallLog(taskID, "✅ "+line[8:])
		} else if strings.HasPrefix(line, "DETAIL:") {
			detail := line[7:]
			if detail != "" && !strings.HasPrefix(detail, "Get:") && !strings.HasPrefix(detail, "Hit:") {
				addInstallLog(taskID, detail)
			}
		} else if line != "" {
			addInstallLog(taskID, line)
		}
	}

	err = cmd.Wait()

	if err != nil {
		setProgress(taskID, "failed", 100, "安装失败: "+err.Error())
		addLog("error", "installer", service+" 安装失败")
	} else {
		setProgress(taskID, "success", 100, service+" 安装成功！")
		updateDBInstance(service)
	}
}

func relatedLog(taskID, service string) {
	// 记录到系统日志
	addLog("error", "installer", service+" 安装失败")
}

func updateDBInstance(service string) {
	// 更新数据库实例状态
	switch service {
	case "mysql":
		config.DB.Model(&model.DBInstance{}).Where("type = ?", "mysql").Updates(map[string]interface{}{
			"status":  "running",
			"version": getMySQLVersion(),
		})
	case "postgresql":
		config.DB.Model(&model.DBInstance{}).Where("type = ?", "postgresql").Updates(map[string]interface{}{
			"status":  "running",
			"version": getPostgreSQLVersion(),
		})
	case "redis":
		config.DB.Model(&model.DBInstance{}).Where("type = ?", "redis").Updates(map[string]interface{}{
			"status":  "running",
			"version": getRedisVersion(),
		})
	}
	addLog("info", "installer", service+" 安装成功")
}

// ========== 安装入口 ==========

// GetAvailableVersions 获取可用版本列表
func GetAvailableVersions(c *gin.Context) {
	serviceType := c.Param("type")

	scriptPath := "./scripts/install.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = "/item/OpsManage/backend/scripts/install.sh"
	}

	cmd := exec.Command(scriptPath, serviceType, "versions")
	output, err := cmd.Output()
	if err != nil {
		success(c, []string{"latest"})
		return
	}

	versions := []string{"latest"}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "===") && !strings.HasPrefix(line, "INFO:") {
			// 去掉版本号后的后缀（如 -1ubuntu1）
			parts := strings.Split(line, "-")
			if len(parts) > 0 {
				version := parts[0]
				if !contains(versions, version) {
					versions = append(versions, version)
				}
			}
		}
	}

	success(c, versions)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// NginxInstall 安装 Nginx
func NginxInstall(c *gin.Context) {
	if isNginxInstalled() {
		fail(c, 400, "Nginx 已安装")
		return
	}

	var req struct {
		Version string `json:"version"`
	}
	c.ShouldBindJSON(&req)

	taskID := fmt.Sprintf("nginx-%d", time.Now().Unix())
	createProgress(taskID, "nginx")

	args := []string{}
	if req.Version != "" && req.Version != "latest" {
		args = append(args, req.Version)
	}

	go runInstallScript(taskID, "nginx", args...)

	success(c, gin.H{"task_id": taskID, "message": "正在安装 Nginx..."})
}

// MySQLInstall 安装 MySQL
func MySQLInstall(c *gin.Context) {
	if isMySQLInstalled() {
		fail(c, 400, "MySQL 已安装")
		return
	}

	var req struct {
		Version string `json:"version"`
		RootPass string `json:"root_pass"`
	}
	c.ShouldBindJSON(&req)

	taskID := fmt.Sprintf("mysql-%d", time.Now().Unix())
	createProgress(taskID, "mysql")

	args := []string{}
	if req.Version != "" && req.Version != "latest" {
		args = append(args, req.Version)
	}
	if req.RootPass != "" {
		args = append(args, req.RootPass)
	}

	go runInstallScript(taskID, "mysql", args...)

	success(c, gin.H{"task_id": taskID, "message": "正在安装 MySQL..."})
}

// PostgreSQLInstall 安装 PostgreSQL
func PostgreSQLInstall(c *gin.Context) {
	if isPostgreSQLInstalled() {
		fail(c, 400, "PostgreSQL 已安装")
		return
	}

	var req struct {
		Version string `json:"version"`
	}
	c.ShouldBindJSON(&req)

	taskID := fmt.Sprintf("pg-%d", time.Now().Unix())
	createProgress(taskID, "postgresql")

	args := []string{}
	if req.Version != "" && req.Version != "latest" {
		args = append(args, req.Version)
	}

	go runInstallScript(taskID, "postgresql", args...)

	success(c, gin.H{"task_id": taskID, "message": "正在安装 PostgreSQL..."})
}

// RedisInstall 安装 Redis
func RedisInstall(c *gin.Context) {
	if isRedisInstalled() {
		fail(c, 400, "Redis 已安装")
		return
	}

	var req struct {
		Version string `json:"version"`
	}
	c.ShouldBindJSON(&req)

	taskID := fmt.Sprintf("redis-%d", time.Now().Unix())
	createProgress(taskID, "redis")

	args := []string{}
	if req.Version != "" && req.Version != "latest" {
		args = append(args, req.Version)
	}

	go runInstallScript(taskID, "redis", args...)

	success(c, gin.H{"task_id": taskID, "message": "正在安装 Redis..."})
}

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
