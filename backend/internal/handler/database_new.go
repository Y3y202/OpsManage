package handler

import (
	"database/sql"
	"fmt"
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// ========== 数据库服务管理 ==========

// DBServiceStatus 获取所有数据库服务状态
func DBServiceStatus(c *gin.Context) {
	services := []map[string]interface{}{
		{
			"type":    "nginx",
			"name":    "Nginx",
			"installed": isNginxInstalled(),
			"running":   isNginxRunning(),
			"version":   getNginxVersion(),
			"port":      80,
		},
		{
			"type":      "mysql",
			"name":      "MySQL",
			"installed": isMySQLInstalled(),
			"running":   isMySQLRunning(),
			"version":   getMySQLVersion(),
			"port":      3306,
		},
		{
			"type":      "postgresql",
			"name":      "PostgreSQL",
			"installed": isPostgreSQLInstalled(),
			"running":   isPostgreSQLRunning(),
			"version":   getPostgreSQLVersion(),
			"port":      5432,
		},
		{
			"type":      "redis",
			"name":      "Redis",
			"installed": isRedisInstalled(),
			"running":   isRedisRunning(),
			"version":   getRedisVersion(),
			"port":      6379,
		},
	}
	success(c, services)
}

// ListDBInstances 获取数据库实例列表
func ListDBInstances(c *gin.Context) {
	var instances []model.DBInstance
	var total int64
	query := config.DB.Model(&model.DBInstance{})
	query.Count(&total)
	paginate(c, query).Order("id desc").Find(&instances)
	pageResult(c, instances, total)
}

// CreateDBInstance 创建数据库实例
func CreateDBInstance(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		Type       string `json:"type" binding:"required"`
		Version    string `json:"version"`
		InstallWay string `json:"install_way"`
		RootPass   string `json:"root_pass"`
		Port       int    `json:"port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	if req.InstallWay == "" {
		req.InstallWay = "apt"
	}

	// 设置默认端口和配置路径
	var port int
	var configPath, dataPath string
	switch req.Type {
	case "mysql":
		port = 3306
		configPath = "/etc/mysql/mysql.conf.d/mysqld.cnf"
		dataPath = "/var/lib/mysql"
	case "postgresql":
		port = 5432
		configPath = "/etc/postgresql/*/main/postgresql.conf"
		dataPath = "/var/lib/postgresql"
	case "redis":
		port = 6379
		configPath = "/etc/redis/redis.conf"
		dataPath = "/var/lib/redis"
	default:
		fail(c, 400, "不支持的数据库类型")
		return
	}
	if req.Port > 0 {
		port = req.Port
	}

	instance := model.DBInstance{
		Name:       req.Name,
		Type:       req.Type,
		Version:    req.Version,
		InstallWay: req.InstallWay,
		Host:       "127.0.0.1",
		Port:       port,
		RootPass:   req.RootPass,
		Status:     "stopped",
		ConfigPath: configPath,
		DataPath:   dataPath,
	}

	// 检测现有安装
	switch req.Type {
	case "mysql":
		if isMySQLInstalled() {
			instance.Status = "running"
			instance.Version = getMySQLVersion()
		}
	case "postgresql":
		if isPostgreSQLInstalled() {
			instance.Status = "running"
			instance.Version = getPostgreSQLVersion()
		}
	case "redis":
		if isRedisInstalled() {
			instance.Status = "running"
			instance.Version = getRedisVersion()
		}
	}

	if err := config.DB.Create(&instance).Error; err != nil {
		fail(c, 500, "创建失败")
		return
	}

	addLog("info", "database", "创建数据库实例: "+instance.Name+" ("+instance.Type+")")
	success(c, instance)
}

// GetDBInstance 获取实例详情
func GetDBInstance(c *gin.Context) {
	id := c.Param("id")
	var instance model.DBInstance
	if err := config.DB.First(&instance, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	// 获取实例状态
	info := map[string]interface{}{
		"instance": instance,
	}

	// 获取数据库数量
	var dbCount int64
	config.DB.Model(&model.DBDatabase{}).Where("instance_id = ?", instance.ID).Count(&dbCount)
	info["db_count"] = dbCount

	// 获取用户数量
	var userCount int64
	config.DB.Model(&model.DBUser{}).Where("instance_id = ?", instance.ID).Count(&userCount)
	info["user_count"] = userCount

	// 获取备份数量
	var backupCount int64
	config.DB.Model(&model.DBBackup{}).Where("instance_id = ?", instance.ID).Count(&backupCount)
	info["backup_count"] = backupCount

	success(c, info)
}

// DBInstanceAction 数据库实例操作（安装/启动/停止/重启）
func DBInstanceAction(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")

	var instance model.DBInstance
	if err := config.DB.First(&instance, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var cmd *exec.Cmd
	var output []byte
	var err error

	switch action {
	case "install":
		go func() {
			addLog("info", "database", "开始安装 "+instance.Type+"...")
			switch instance.Type {
			case "mysql":
				installMySQL(instance.RootPass)
			case "postgresql":
				installPostgreSQL()
			case "redis":
				installRedis()
			}
			config.DB.Model(&instance).Updates(map[string]interface{}{
				"status":  "running",
				"version": getDBVersion(instance.Type),
			})
			addLog("info", "database", instance.Type+" 安装完成")
		}()
		success(c, gin.H{"message": "正在安装 " + instance.Type + "..."})
		return

	case "start":
		cmd = getServiceCommand(instance.Type, "start")
	case "stop":
		cmd = getServiceCommand(instance.Type, "stop")
	case "restart":
		cmd = getServiceCommand(instance.Type, "restart")
	default:
		fail(c, 400, "不支持的操作")
		return
	}

	output, err = cmd.CombinedOutput()
	if err != nil {
		fail(c, 500, "操作失败: "+string(output))
		return
	}

	status := isDBRunning(instance.Type)
	config.DB.Model(&instance).Update("status", boolToStatus(status))
	success(c, gin.H{"status": boolToStatus(status), "message": "操作成功"})
}

// DBInstanceConfig 获取/编辑数据库配置
func DBInstanceConfig(c *gin.Context) {
	id := c.Param("id")
	var instance model.DBInstance
	if err := config.DB.First(&instance, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	if c.Request.Method == "GET" {
		content := ""
		if instance.ConfigPath != "" {
			// 处理通配符路径
			paths, _ := filepath.Glob(instance.ConfigPath)
			if len(paths) > 0 {
				data, err := os.ReadFile(paths[0])
				if err == nil {
					content = string(data)
				}
			}
		}
		success(c, gin.H{"config_path": instance.ConfigPath, "content": content})
		return
	}

	// POST - 保存配置
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	configPath := instance.ConfigPath
	paths, _ := filepath.Glob(configPath)
	if len(paths) > 0 {
		configPath = paths[0]
	}

	if err := os.WriteFile(configPath, []byte(req.Content), 0644); err != nil {
		fail(c, 500, "写入配置失败")
		return
	}

	addLog("info", "database", "编辑数据库配置: "+instance.Type)
	success(c, gin.H{"message": "配置已保存，请重启数据库服务使配置生效"})
}

// ========== 数据库管理 ==========

// ListDBDatabases 获取数据库列表
func ListDBDatabases(c *gin.Context) {
	instanceID := c.Param("instance_id")
	var databases []model.DBDatabase
	var total int64
	query := config.DB.Model(&model.DBDatabase{}).Where("instance_id = ?", instanceID)
	query.Count(&total)
	query.Order("id desc").Find(&databases)

	// 如果数据库为空，尝试从实际数据库同步
	if total == 0 {
		iid, _ := strconv.ParseUint(instanceID, 10, 64)
		syncDatabasesFromService(uint(iid))
		query.Count(&total)
		query.Order("id desc").Find(&databases)
	}

	pageResult(c, databases, total)
}

// CreateDBDatabase 创建数据库
func CreateDBDatabase(c *gin.Context) {
	var req struct {
		InstanceID uint   `json:"instance_id" binding:"required"`
		Name       string `json:"name" binding:"required"`
		Charset    string `json:"charset"`
		Collation  string `json:"collation"`
		Remark     string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	var instance model.DBInstance
	if err := config.DB.First(&instance, req.InstanceID).Error; err != nil {
		fail(c, 404, "数据库实例不存在")
		return
	}

	if req.Charset == "" {
		req.Charset = "utf8mb4"
	}

	// 在实际数据库中创建
	err := createDatabaseOnService(&instance, req.Name, req.Charset, req.Collation)
	if err != nil {
		fail(c, 500, "创建数据库失败: "+err.Error())
		return
	}

	db := model.DBDatabase{
		InstanceID: req.InstanceID,
		Name:       req.Name,
		Charset:    req.Charset,
		Collation:  req.Collation,
		Remark:     req.Remark,
	}

	config.DB.Create(&db)
	addLog("info", "database", fmt.Sprintf("创建数据库: %s (实例: %s)", req.Name, instance.Name))
	success(c, db)
}

// DeleteDBDatabase 删除数据库
func DeleteDBDatabase(c *gin.Context) {
	id := c.Param("did")
	var db model.DBDatabase
	if err := config.DB.First(&db, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var instance model.DBInstance
	config.DB.First(&instance, db.InstanceID)

	// 在实际数据库中删除
	err := dropDatabaseOnService(&instance, db.Name)
	if err != nil {
		fail(c, 500, "删除数据库失败: "+err.Error())
		return
	}

	config.DB.Delete(&db)
	addLog("info", "database", fmt.Sprintf("删除数据库: %s", db.Name))
	success(c, nil)
}

// SyncDBDatabases 同步数据库列表
func SyncDBDatabases(c *gin.Context) {
	instanceID := c.Param("instance_id")
	iid, _ := strconv.ParseUint(instanceID, 10, 64)
	syncDatabasesFromService(uint(iid))
	ListDBDatabases(c)
}

// ========== 用户管理 ==========

// ListDBUsers 获取数据库用户列表
func ListDBUsers(c *gin.Context) {
	instanceID := c.Param("instance_id")
	var users []model.DBUser
	config.DB.Where("instance_id = ?", instanceID).Find(&users)
	success(c, users)
}

// CreateDBUser 创建数据库用户
func CreateDBUser(c *gin.Context) {
	var req struct {
		InstanceID uint   `json:"instance_id" binding:"required"`
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		Host       string `json:"host"`
		DBName     string `json:"db_name"`
		Privileges string `json:"privileges"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	var instance model.DBInstance
	if err := config.DB.First(&instance, req.InstanceID).Error; err != nil {
		fail(c, 404, "数据库实例不存在")
		return
	}

	if req.Host == "" {
		req.Host = "%"
	}
	if req.Privileges == "" {
		req.Privileges = "ALL"
	}

	// 在实际数据库中创建用户
	err := createDBUserOnService(&instance, req.Username, req.Password, req.Host, req.DBName, req.Privileges)
	if err != nil {
		fail(c, 500, "创建用户失败: "+err.Error())
		return
	}

	user := model.DBUser{
		InstanceID: req.InstanceID,
		Username:   req.Username,
		Host:       req.Host,
		DBName:     req.DBName,
		Privileges: req.Privileges,
	}

	config.DB.Create(&user)
	addLog("info", "database", fmt.Sprintf("创建数据库用户: %s@%s", req.Username, req.Host))
	success(c, user)
}

// DeleteDBUser 删除数据库用户
func DeleteDBUser(c *gin.Context) {
	id := c.Param("did")
	var user model.DBUser
	if err := config.DB.First(&user, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var instance model.DBInstance
	config.DB.First(&instance, user.InstanceID)

	dropDBUserOnService(&instance, user.Username, user.Host)
	config.DB.Delete(&user)
	addLog("info", "database", fmt.Sprintf("删除数据库用户: %s@%s", user.Username, user.Host))
	success(c, nil)
}

// ========== 备份管理 ==========

// ListDBBackups 获取备份列表
func ListDBBackups(c *gin.Context) {
	instanceID := c.Param("instance_id")
	var backups []model.DBBackup
	config.DB.Where("instance_id = ?", instanceID).Order("id desc").Find(&backups)
	success(c, backups)
}

// CreateDBBackup 创建备份
func CreateDBBackup(c *gin.Context) {
	var req struct {
		InstanceID uint   `json:"instance_id" binding:"required"`
		DBName     string `json:"db_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	var instance model.DBInstance
	if err := config.DB.First(&instance, req.InstanceID).Error; err != nil {
		fail(c, 404, "数据库实例不存在")
		return
	}

	go func() {
		backupDir := fmt.Sprintf("/var/backups/opsmanage/%s", instance.Type)
		os.MkdirAll(backupDir, 0755)
		fileName := fmt.Sprintf("%s_%s.sql", req.DBName, time.Now().Format("20060102_150405"))
		filePath := fmt.Sprintf("%s/%s", backupDir, fileName)

		var cmd *exec.Cmd
		switch instance.Type {
		case "mysql":
			cmd = exec.Command("mysqldump", "-h", instance.Host, "-P", strconv.Itoa(instance.Port),
				"-u", "root", "--result-file="+filePath, req.DBName)
		case "postgresql":
			cmd = exec.Command("pg_dump", "-h", instance.Host, "-p", strconv.Itoa(instance.Port),
				"-U", "postgres", "-f", filePath, req.DBName)
		}

		output, err := cmd.CombinedOutput()
		status := "success"
		if err != nil {
			status = "failed"
			addLog("error", "database", "备份失败: "+string(output))
		}

		info, _ := os.Stat(filePath)
		var size int64
		if info != nil {
			size = info.Size()
		}

		backup := model.DBBackup{
			InstanceID: req.InstanceID,
			DBName:     req.DBName,
			FilePath:   filePath,
			Size:       size,
			Status:     status,
		}
		config.DB.Create(&backup)
		addLog("info", "database", fmt.Sprintf("备份数据库: %s -> %s", req.DBName, filePath))
	}()

	success(c, gin.H{"message": "正在备份..."})
}

// RestoreDBBackup 恢复备份
func RestoreDBBackup(c *gin.Context) {
	id := c.Param("bid")
	var backup model.DBBackup
	if err := config.DB.First(&backup, id).Error; err != nil {
		fail(c, 404, "备份不存在")
		return
	}

	var instance model.DBInstance
	config.DB.First(&instance, backup.InstanceID)

	go func() {
		var cmd *exec.Cmd
		switch instance.Type {
		case "mysql":
			cmd = exec.Command("mysql", "-h", instance.Host, "-P", strconv.Itoa(instance.Port),
				"-u", "root", backup.DBName)
			stdin, _ := cmd.StdinPipe()
			go func() {
				data, _ := os.ReadFile(backup.FilePath)
				stdin.Write(data)
				stdin.Close()
			}()
		case "postgresql":
			cmd = exec.Command("psql", "-h", instance.Host, "-p", strconv.Itoa(instance.Port),
				"-U", "postgres", "-d", backup.DBName, "-f", backup.FilePath)
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			addLog("error", "database", "恢复失败: "+string(output))
		} else {
			addLog("info", "database", fmt.Sprintf("恢复数据库: %s <- %s", backup.DBName, backup.FilePath))
		}
	}()

	success(c, gin.H{"message": "正在恢复..."})
}

// ========== 数据库实时状态 ==========

// DBInstanceStats 获取数据库实时统计
func DBInstanceStats(c *gin.Context) {
	id := c.Param("id")
	var instance model.DBInstance
	if err := config.DB.First(&instance, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	stats := map[string]interface{}{
		"running": isDBRunning(instance.Type),
	}

	switch instance.Type {
	case "mysql":
		stats["connections"] = getMySQLConnections()
		stats["queries_per_sec"] = getMySQLQueriesPerSec()
		stats["uptime"] = getMySQLUptime()
	case "postgresql":
		stats["connections"] = getPGConnections(instance.Host, instance.Port)
	case "redis":
		info := getRedisInfo()
		stats["connected_clients"] = info["connected_clients"]
		stats["used_memory"] = info["used_memory_human"]
		stats["uptime"] = info["uptime_in_seconds"]
	}

	success(c, stats)
}

// ========== 辅助函数 ==========

func isMySQLInstalled() bool {
	_, err := exec.LookPath("mysqld")
	return err == nil
}

func isMySQLRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "mysql")
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		return true
	}
	cmd = exec.Command("systemctl", "is-active", "mysqld")
	output, err = cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) == "active"
}

func getMySQLVersion() string {
	cmd := exec.Command("mysql", "--version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`Ver\s+([\d.]+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func isPostgreSQLInstalled() bool {
	_, err := exec.LookPath("psql")
	return err == nil
}

func isPostgreSQLRunning() bool {
	cmd := exec.Command("pg_isready")
	err := cmd.Run()
	return err == nil
}

func getPostgreSQLVersion() string {
	cmd := exec.Command("psql", "--version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`([\d.]+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func isRedisInstalled() bool {
	_, err := exec.LookPath("redis-server")
	return err == nil
}

func isRedisRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "redis-server")
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("systemctl", "is-active", "redis")
		output, err = cmd.Output()
	}
	return err == nil && strings.TrimSpace(string(output)) == "active"
}

func getRedisVersion() string {
	cmd := exec.Command("redis-server", "--version")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`v=([\d.]+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func isDBRunning(dbType string) bool {
	switch dbType {
	case "mysql":
		return isMySQLRunning()
	case "postgresql":
		return isPostgreSQLRunning()
	case "redis":
		return isRedisRunning()
	}
	return false
}

func getDBVersion(dbType string) string {
	switch dbType {
	case "mysql":
		return getMySQLVersion()
	case "postgresql":
		return getPostgreSQLVersion()
	case "redis":
		return getRedisVersion()
	}
	return ""
}

func getServiceCommand(dbType, action string) *exec.Cmd {
	switch dbType {
	case "mysql":
		return exec.Command("systemctl", action, "mysql")
	case "postgresql":
		return exec.Command("systemctl", action, "postgresql")
	case "redis":
		return exec.Command("systemctl", action, "redis-server")
	}
	return exec.Command("echo", "unsupported")
}

func boolToStatus(b bool) string {
	if b {
		return "running"
	}
	return "stopped"
}

func installMySQL(rootPass string) {
	exec.Command("bash", "-c", "apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq mysql-server").Run()
	exec.Command("systemctl", "enable", "mysql").Run()
	exec.Command("systemctl", "start", "mysql").Run()
	if rootPass != "" {
		exec.Command("mysql", "-e", fmt.Sprintf("ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '%s';", rootPass)).Run()
	}
}

func installPostgreSQL() {
	exec.Command("bash", "-c", "apt-get update -qq && apt-get install -y -qq postgresql postgresql-contrib").Run()
	exec.Command("systemctl", "enable", "postgresql").Run()
	exec.Command("systemctl", "start", "postgresql").Run()
}

func installRedis() {
	exec.Command("bash", "-c", "apt-get update -qq && apt-get install -y -qq redis-server").Run()
	exec.Command("systemctl", "enable", "redis-server").Run()
	exec.Command("systemctl", "start", "redis-server").Run()
}

func createDatabaseOnService(instance *model.DBInstance, name, charset, collation string) error {
	switch instance.Type {
	case "mysql":
		cmd := exec.Command("mysql", "-h", instance.Host, "-P", strconv.Itoa(instance.Port), "-u", "root", "-e",
			fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s;", name, charset, collation))
		return cmd.Run()
	case "postgresql":
		cmd := exec.Command("createdb", "-h", instance.Host, "-p", strconv.Itoa(instance.Port), "-U", "postgres", name)
		return cmd.Run()
	}
	return fmt.Errorf("不支持的数据库类型")
}

func dropDatabaseOnService(instance *model.DBInstance, name string) error {
	switch instance.Type {
	case "mysql":
		cmd := exec.Command("mysql", "-h", instance.Host, "-P", strconv.Itoa(instance.Port), "-u", "root", "-e",
			fmt.Sprintf("DROP DATABASE `%s`;", name))
		return cmd.Run()
	case "postgresql":
		cmd := exec.Command("dropdb", "-h", instance.Host, "-p", strconv.Itoa(instance.Port), "-U", "postgres", name)
		return cmd.Run()
	}
	return fmt.Errorf("不支持的数据库类型")
}

func syncDatabasesFromService(instanceID uint) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, instanceID).Error; err != nil {
		return
	}

	var dbNames []string
	switch instance.Type {
	case "mysql":
		cmd := exec.Command("mysql", "-h", instance.Host, "-P", strconv.Itoa(instance.Port), "-u", "root", "-N", "-e", "SHOW DATABASES;")
		output, err := cmd.Output()
		if err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				name := strings.TrimSpace(line)
				if name != "" && name != "information_schema" && name != "performance_schema" && name != "mysql" && name != "sys" {
					dbNames = append(dbNames, name)
				}
			}
		}
	case "postgresql":
		cmd := exec.Command("psql", "-h", instance.Host, "-p", strconv.Itoa(instance.Port), "-U", "postgres", "-t", "-c", "SELECT datname FROM pg_database WHERE datistemplate = false;")
		output, err := cmd.Output()
		if err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				name := strings.TrimSpace(line)
				if name != "" && name != "postgres" {
					dbNames = append(dbNames, name)
				}
			}
		}
	}

	// 保存到数据库
	for _, name := range dbNames {
		var count int64
		config.DB.Model(&model.DBDatabase{}).Where("instance_id = ? AND name = ?", instanceID, name).Count(&count)
		if count == 0 {
			config.DB.Create(&model.DBDatabase{
				InstanceID: instanceID,
				Name:       name,
				Charset:    "utf8mb4",
			})
		}
	}
}

func createDBUserOnService(instance *model.DBInstance, username, password, host, dbName, privileges string) error {
	switch instance.Type {
	case "mysql":
		createSQL := fmt.Sprintf("CREATE USER '%s'@'%s' IDENTIFIED BY '%s';", username, host, password)
		grantSQL := fmt.Sprintf("GRANT %s ON `%s`.* TO '%s'@'%s';", privileges, dbName, username, host)
		cmd := exec.Command("mysql", "-h", instance.Host, "-P", strconv.Itoa(instance.Port), "-u", "root", "-e", createSQL+grantSQL+" FLUSH PRIVILEGES;")
		return cmd.Run()
	case "postgresql":
		cmd := exec.Command("psql", "-h", instance.Host, "-p", strconv.Itoa(instance.Port), "-U", "postgres", "-c",
			fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s';", username, password))
		cmd.Run()
		if dbName != "" {
			cmd = exec.Command("psql", "-h", instance.Host, "-p", strconv.Itoa(instance.Port), "-U", "postgres", "-c",
				fmt.Sprintf("GRANT %s ON DATABASE %s TO %s;", privileges, dbName, username))
			return cmd.Run()
		}
	}
	return nil
}

func dropDBUserOnService(instance *model.DBInstance, username, host string) {
	switch instance.Type {
	case "mysql":
		exec.Command("mysql", "-h", instance.Host, "-P", strconv.Itoa(instance.Port), "-u", "root", "-e",
			fmt.Sprintf("DROP USER '%s'@'%s';", username, host)).Run()
	case "postgresql":
		exec.Command("dropuser", "-h", instance.Host, "-p", strconv.Itoa(instance.Port), "-U", "postgres", username).Run()
	}
}

func getMySQLConnections() int {
	cmd := exec.Command("mysql", "-u", "root", "-N", "-e", "SHOW STATUS LIKE 'Threads_connected';")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		n, _ := strconv.Atoi(parts[1])
		return n
	}
	return 0
}

func getMySQLQueriesPerSec() float64 {
	cmd := exec.Command("mysql", "-u", "root", "-N", "-e", "SHOW STATUS LIKE 'Queries';")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		n, _ := strconv.ParseFloat(parts[1], 64)
		return n
	}
	return 0
}

func getMySQLUptime() int {
	cmd := exec.Command("mysql", "-u", "root", "-N", "-e", "SHOW STATUS LIKE 'Uptime';")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		n, _ := strconv.Atoi(parts[1])
		return n
	}
	return 0
}

func getPGConnections(host string, port int) int {
	connStr := fmt.Sprintf("host=%s port=%d user=postgres dbname=postgres sslmode=disable", host, port)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return 0
	}
	defer db.Close()

	var count int
	db.QueryRow("SELECT count(*) FROM pg_stat_activity").Scan(&count)
	return count
}

func getRedisInfo() map[string]string {
	cmd := exec.Command("redis-cli", "INFO")
	output, err := cmd.Output()
	if err != nil {
		return map[string]string{}
	}

	info := map[string]string{}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return info
}
