package handler

import (
	"fmt"
	"net/http"
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ok 返回统一格式的成功响应
func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data, "msg": "操作成功"})
}

// okMsg 返回带消息的成功响应
func okMsg(c *gin.Context, data interface{}, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": data, "msg": msg})
}

// ========== 数据库服务管理 ==========

// detectDBVersion 检测已安装的数据库版本
func detectDBVersion(dbType string) string {
	var cmd *exec.Cmd
	switch dbType {
	case "mysql":
		cmd = exec.Command("mysql", "--version")
	case "postgresql":
		cmd = exec.Command("psql", "--version")
	case "redis":
		cmd = exec.Command("redis-cli", "--version")
	default:
		return ""
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	s := string(out)
	// Parse version from output
	switch dbType {
	case "mysql":
		// "mysql  Ver 8.0.45 for Linux on x86_64 ..."
		parts := strings.Fields(s)
		for i, p := range parts {
			if p == "Ver" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	case "postgresql":
		// "psql (PostgreSQL) 16.3"
		parts := strings.Fields(s)
		for _, p := range parts {
			if len(p) > 2 && (p[0] >= '0' && p[0] <= '9') {
				return strings.Trim(p, "()")
			}
		}
	case "redis":
		// "redis-cli 7.0.15"
		parts := strings.Fields(s)
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

// isDBRunning 检查数据库服务是否在运行
func isDBRunning(dbType string) bool {
	var cmd *exec.Cmd
	switch dbType {
	case "mysql":
		cmd = exec.Command("systemctl", "is-active", "mysql")
	case "postgresql":
		cmd = exec.Command("systemctl", "is-active", "postgresql")
	case "redis":
		cmd = exec.Command("systemctl", "is-active", "redis-server")
	default:
		return false
	}
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "active"
}

// getDBDefaultPort 获取数据库默认端口
func getDBDefaultPort(dbType string) int {
	switch dbType {
	case "mysql":
		return 3306
	case "postgresql":
		return 5432
	case "redis":
		return 6379
	default:
		return 0
	}
}

// getDBConfigPath 获取数据库配置文件路径
func getDBConfigPath(dbType string) string {
	switch dbType {
	case "mysql":
		return "/etc/mysql/mysql.conf.d/mysqld.cnf"
	case "postgresql":
		// Find the active postgresql.conf
		cmd := exec.Command("sudo", "-u", "postgres", "psql", "-t", "-c",
			"SHOW config_file;")
		out, err := cmd.Output()
		if err == nil {
			p := strings.TrimSpace(string(out))
			if p != "" {
				return p
			}
		}
		return "/etc/postgresql/16/main/postgresql.conf"
	case "redis":
		return "/etc/redis/redis.conf"
	default:
		return ""
	}
}

// DBServiceStatus 返回三个数据库服务的状态
// GET /api/databases/services/status
func DBServiceStatus(c *gin.Context) {
	dbTypes := []string{"mysql", "postgresql", "redis"}
	names := map[string]string{
		"mysql":      "MySQL",
		"postgresql": "PostgreSQL",
		"redis":      "Redis",
	}

	// Get instances from DB to find stored root passwords
	var instances []model.DBInstance
	config.DB.Find(&instances)
	instMap := make(map[string]model.DBInstance)
	for _, inst := range instances {
		instMap[inst.Type] = inst
	}

	var services []gin.H
	for _, dbType := range dbTypes {
		version := detectDBVersion(dbType)
		installed := version != ""
		running := isDBRunning(dbType)

		svc := gin.H{
			"type":     dbType,
			"name":     names[dbType],
			"version":  version,
			"installed": installed,
			"running":  running,
			"port":     getDBDefaultPort(dbType),
		}
		if inst, ok := instMap[dbType]; ok {
			svc["instance_id"] = inst.ID
			svc["root_pass"] = inst.RootPass
		}
		services = append(services, svc)
	}

	ok(c, services)
}

// ListDBInstances 列出所有数据库实例
// GET /api/databases/instances
func ListDBInstances(c *gin.Context) {
	var instances []model.DBInstance
	config.DB.Find(&instances)

	// Update status from real system
	for i := range instances {
		instances[i].Version = detectDBVersion(instances[i].Type)
		if isDBRunning(instances[i].Type) {
			instances[i].Status = "running"
		} else {
			instances[i].Status = "stopped"
		}
	}

	ok(c, gin.H{"items": instances})
}

// CreateDBInstance 创建数据库实例
// POST /api/databases/instances
func CreateDBInstance(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		Type       string `json:"type" binding:"required"`
		Version    string `json:"version"`
		Port       int    `json:"port"`
		RootPass   string `json:"root_pass"`
		InstallWay string `json:"install_way"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, err.Error())
		return
	}

	// Check if instance of this type already exists
	var existing model.DBInstance
	if err := config.DB.Where("type = ?", req.Type).First(&existing).Error; err == nil {
		fail(c, 409, fmt.Sprintf("%s 实例已存在", req.Type))
		return
	}

	version := req.Version
	if version == "" {
		version = detectDBVersion(req.Type)
	}

	status := "stopped"
	if isDBRunning(req.Type) {
		status = "running"
	}

	port := req.Port
	if port == 0 {
		port = getDBDefaultPort(req.Type)
	}

	installWay := req.InstallWay
	if installWay == "" {
		installWay = "apt"
	}

	instance := model.DBInstance{
		Name:       req.Name,
		Type:       req.Type,
		Version:    version,
		InstallWay: installWay,
		Host:       "127.0.0.1",
		Port:       port,
		RootPass:   req.RootPass,
		Status:     status,
		ConfigPath: getDBConfigPath(req.Type),
		Remark:     "",
	}

	if err := config.DB.Create(&instance).Error; err != nil {
		fail(c, 500, "创建失败: "+err.Error())
		return
	}

	ok(c, instance)
}

// GetDBInstance 获取单个实例
// GET /api/databases/instances/:id
func GetDBInstance(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}
	instance.Version = detectDBVersion(instance.Type)
	if isDBRunning(instance.Type) {
		instance.Status = "running"
	} else {
		instance.Status = "stopped"
	}
	c.JSON(http.StatusOK, instance)
}

// DBInstanceAction 对实例执行操作 (start/stop/restart/install)
// POST /api/databases/instances/:id/:action
func DBInstanceAction(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}

	action := c.Param("action")
	var serviceName string
	switch instance.Type {
	case "mysql":
		serviceName = "mysql"
	case "postgresql":
		serviceName = "postgresql"
	case "redis":
		serviceName = "redis-server"
	default:
		fail(c, 400, "不支持的数据库类型")
		return
	}

	var cmd *exec.Cmd
	switch action {
	case "start":
		cmd = exec.Command("systemctl", "start", serviceName)
	case "stop":
		cmd = exec.Command("systemctl", "stop", serviceName)
	case "restart":
		cmd = exec.Command("systemctl", "restart", serviceName)
	case "install":
		// Trigger install via installer handler
		okMsg(c, nil, "请使用安装脚本安装")
		return
	default:
		fail(c, 400, "不支持的操作: "+action)
		return
	}

	if err := cmd.Run(); err != nil {
		fail(c, 500, fmt.Sprintf("%s 失败: %v", action, err))
		return
	}

	// Update status
	time.Sleep(500 * time.Millisecond)
	if isDBRunning(instance.Type) {
		instance.Status = "running"
	} else {
		instance.Status = "stopped"
	}
	instance.Version = detectDBVersion(instance.Type)
	config.DB.Save(&instance)

	okMsg(c, gin.H{"status": instance.Status}, action+" 成功")
}

// DBInstanceConfig 读取/保存实例配置
// GET/PUT /api/databases/instances/:id/config
func DBInstanceConfig(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}

	configPath := instance.ConfigPath
	if configPath == "" {
		configPath = getDBConfigPath(instance.Type)
	}

	if c.Request.Method == "GET" {
		content := ""
		if configPath != "" {
			cmd := exec.Command("cat", configPath)
			out, err := cmd.Output()
			if err == nil {
				content = string(out)
			}
		}
		ok(c, gin.H{
			"config_path": configPath,
			"content":     content,
		})
		return
	}

	// PUT - save config
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, err.Error())
		return
	}

	// Write config file
	cmd := exec.Command("sh", "-c", fmt.Sprintf("echo %s > %s",
		shellQuote(req.Content), shellQuote(configPath)))
	if err := cmd.Run(); err != nil {
		fail(c, 500, "保存配置失败: "+err.Error())
		return
	}

	okMsg(c, nil, "配置已保存")
}

// DBInstanceStats 获取实例统计
// GET /api/databases/instances/:id/stats
func DBInstanceStats(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}

	stats := gin.H{
		"status":  instance.Status,
		"version": detectDBVersion(instance.Type),
	}

	switch instance.Type {
	case "mysql":
		if out, err := runMySQL("SHOW GLOBAL STATUS LIKE 'Threads_connected';"); err == nil {
			parts := strings.Fields(strings.TrimSpace(out))
			if len(parts) >= 2 {
				stats["connections"] = parts[1]
			}
		}
		if out, err := runMySQL("SHOW GLOBAL STATUS LIKE 'Uptime';"); err == nil {
			parts := strings.Fields(strings.TrimSpace(out))
			if len(parts) >= 2 {
				stats["uptime"] = parts[1]
			}
		}
	case "postgresql":
		if out, err := runPSQL("SELECT count(*) FROM pg_stat_activity;"); err == nil {
			stats["connections"] = strings.TrimSpace(out)
		}
	case "redis":
		if out, err := exec.Command("redis-cli", "info", "server").Output(); err == nil {
			stats["info"] = string(out)
		}
	}

	ok(c, stats)
}

// ========== 数据库管理 ==========

// runMySQL 执行 MySQL 命令
func runMySQL(query string) (string, error) {
	// Try with stored password first
	var instance model.DBInstance
	config.DB.Where("type = ?", "mysql").First(&instance)

	var cmd *exec.Cmd
	if instance.RootPass != "" {
		cmd = exec.Command("mysql", "-uroot", "-p"+instance.RootPass, "-N", "-e", query)
	} else {
		cmd = exec.Command("mysql", "-uroot", "-N", "-e", query)
	}
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

// runMySQLExec 执行 MySQL 命令并返回 error
func runMySQLExec(query string) error {
	_, err := runMySQL(query)
	return err
}

// runPSQL 执行 PostgreSQL 命令
func runPSQL(query string) (string, error) {
	cmd := exec.Command("sudo", "-u", "postgres", "psql", "-t", "-A", "-c", query)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

// runPSQLExec 执行 PostgreSQL 命令
func runPSQLExec(query string) error {
	_, err := runPSQL(query)
	return err
}

// ListDBDatabases 列出实例中的所有数据库
// GET /api/databases/instances/:id/databases
func ListDBDatabases(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}

	var dbNames []gin.H

	switch instance.Type {
	case "mysql":
		out, err := runMySQL("SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME NOT IN ('information_schema','mysql','performance_schema','sys') ORDER BY SCHEMA_NAME;")
		if err != nil {
			fail(c, 500, "查询失败: "+err.Error())
			return
		}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) >= 3 {
				// Get size
				sizeOut, _ := runMySQL(fmt.Sprintf(
					"SELECT COALESCE(SUM(data_length+index_length),0) FROM information_schema.TABLES WHERE table_schema='%s';", parts[0]))
				size := int64(0)
				fmt.Sscanf(strings.TrimSpace(sizeOut), "%d", &size)

				dbNames = append(dbNames, gin.H{
					"name":      parts[0],
					"charset":   parts[1],
					"collation": parts[2],
					"size":      size,
				})
			}
		}

	case "postgresql":
		out, err := runPSQL("SELECT datname, pg_encoding_to_char(encoding), datcollate FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres') ORDER BY datname;")
		if err != nil {
			fail(c, 500, "查询失败: "+err.Error())
			return
		}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
			// Get size
			dbName := strings.TrimSpace(parts[0])
			sizeOut, _ := runPSQL(fmt.Sprintf(
				"SELECT pg_database_size('%s');", dbName))
				size := int64(0)
				fmt.Sscanf(strings.TrimSpace(sizeOut), "%d", &size)

				dbNames = append(dbNames, gin.H{
					"name":      strings.TrimSpace(parts[0]),
					"charset":   strings.TrimSpace(parts[1]),
					"collation": strings.TrimSpace(parts[2]),
					"size":      size,
				})
			}
		}

	case "redis":
		// Redis has numbered databases
		out, err := exec.Command("redis-cli", "INFO", "keyspace").Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.HasPrefix(line, "db") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						dbNames = append(dbNames, gin.H{
							"name":    strings.TrimSpace(parts[0]),
							"charset": "-",
							"size":    0,
							"remark":  strings.TrimSpace(parts[1]),
						})
					}
				}
			}
		}
		if len(dbNames) == 0 {
			dbNames = append(dbNames, gin.H{
				"name":    "db0",
				"charset": "-",
				"size":    0,
			})
		}
	}

	ok(c, dbNames)
}

// CreateDBDatabase 在实例中创建新数据库
// POST /api/databases/instances/:id/databases
func CreateDBDatabase(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}

	var req struct {
		Name    string `json:"name" binding:"required"`
		Charset string `json:"charset"`
		Remark  string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, err.Error())
		return
	}

	if req.Charset == "" {
		req.Charset = "utf8mb4"
	}

	var err error
	switch instance.Type {
	case "mysql":
		err = runMySQLExec(fmt.Sprintf(
			"CREATE DATABASE `%s` CHARACTER SET %s;", req.Name, req.Charset))
	case "postgresql":
		err = runPSQLExec(fmt.Sprintf(
			"CREATE DATABASE \"%s\" ENCODING '%s';", req.Name, req.Charset))
	case "redis":
		fail(c, 400, "Redis 不支持创建数据库")
		return
	default:
		fail(c, 400, "不支持的数据库类型")
		return
	}

	if err != nil {
		fail(c, 500, "创建数据库失败: "+err.Error())
		return
	}

	// Save to local DB
	db := model.DBDatabase{
		InstanceID: instance.ID,
		Name:       req.Name,
		Charset:    req.Charset,
		Remark:     req.Remark,
	}
	config.DB.Create(&db)

	ok(c, db)
}

// DeleteDBDatabase 删除数据库
// DELETE /api/databases/databases/:did
func DeleteDBDatabase(c *gin.Context) {
	var db model.DBDatabase
	if err := config.DB.First(&db, c.Param("did")).Error; err != nil {
		fail(c, 404, "记录不存在")
		return
	}

	var instance model.DBInstance
	config.DB.First(&instance, db.InstanceID)

	var err error
	switch instance.Type {
	case "mysql":
		err = runMySQLExec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", db.Name))
	case "postgresql":
		err = runPSQLExec(fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\";", db.Name))
	default:
		fail(c, 400, "不支持的数据库类型")
		return
	}

	if err != nil {
		fail(c, 500, "删除数据库失败: "+err.Error())
		return
	}

	config.DB.Delete(&db)
	okMsg(c, nil, "删除成功")
}

// SyncDBDatabases 同步数据库列表（从真实数据库拉取）
// POST /api/databases/instances/:id/databases/sync
func SyncDBDatabases(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}

	var dbNames []string

	switch instance.Type {
	case "mysql":
		out, err := runMySQL("SHOW DATABASES;")
		if err != nil {
			fail(c, 500, "查询失败: "+err.Error())
			return
		}
		skip := map[string]bool{
			"information_schema": true, "mysql": true,
			"performance_schema": true, "sys": true,
		}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			name := strings.TrimSpace(line)
			if name != "" && !skip[name] {
				dbNames = append(dbNames, name)
			}
		}

	case "postgresql":
		out, err := runPSQL("SELECT datname FROM pg_database WHERE datistemplate = false AND datname != 'postgres';")
		if err != nil {
			fail(c, 500, "查询失败: "+err.Error())
			return
		}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			name := strings.TrimSpace(line)
			if name != "" {
				dbNames = append(dbNames, name)
			}
		}

	case "redis":
		okMsg(c, nil, "Redis 无需同步")
		return
	}

	// Clear old records and insert new
	config.DB.Where("instance_id = ?", instance.ID).Delete(&model.DBDatabase{})

	for _, name := range dbNames {
		db := model.DBDatabase{
			InstanceID: instance.ID,
			Name:       name,
			Charset:    "utf8mb4",
		}
		config.DB.Create(&db)
	}

	okMsg(c, nil, fmt.Sprintf("同步成功，共 %d 个数据库", len(dbNames)))
}

// ========== 用户管理 ==========

// ListDBUsers 列出数据库用户
// GET /api/databases/instances/:id/users
func ListDBUsers(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}

	var users []gin.H

	switch instance.Type {
	case "mysql":
		out, err := runMySQL("SELECT User, Host FROM mysql.user WHERE User NOT IN ('mysql.sys','mysql.session','mysql.infoschema','debian-sys-maint') ORDER BY User;")
		if err != nil {
			fail(c, 500, "查询失败: "+err.Error())
			return
		}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 {
				users = append(users, gin.H{
					"username":   strings.TrimSpace(parts[0]),
					"host":       strings.TrimSpace(parts[1]),
					"db_name":    "*",
					"privileges": "-",
				})
			}
		}

	case "postgresql":
		out, err := runPSQL("SELECT rolname, CASE WHEN rolsuper THEN 'SUPERUSER' WHEN rolcreaterole THEN 'CREATEROLE' ELSE 'USER' END FROM pg_roles WHERE rolname NOT LIKE 'pg_%' ORDER BY rolname;")
		if err != nil {
			fail(c, 500, "查询失败: "+err.Error())
			return
		}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				users = append(users, gin.H{
					"username":   strings.TrimSpace(parts[0]),
					"host":       "*",
					"db_name":    "*",
					"privileges": strings.TrimSpace(parts[1]),
				})
			}
		}

	case "redis":
		out, err := exec.Command("redis-cli", "ACL", "LIST").Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						users = append(users, gin.H{
							"username":   parts[1],
							"host":       "*",
							"db_name":    "*",
							"privileges": strings.Join(parts[2:], " "),
						})
					}
				}
			}
		}
	}

	if users == nil {
		users = []gin.H{}
	}
		ok(c, users)
}

// CreateDBUser 创建数据库用户
// POST /api/databases/instances/:id/users
func CreateDBUser(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}

	var req struct {
		Username   string `json:"username" binding:"required"`
		Password   string `json:"password" binding:"required"`
		Host       string `json:"host"`
		DBName     string `json:"db_name"`
		Privileges string `json:"privileges"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, err.Error())
		return
	}

	if req.Host == "" {
		req.Host = "%"
	}
	if req.Privileges == "" {
		req.Privileges = "ALL"
	}

	var err error
	switch instance.Type {
	case "mysql":
		// Create user
		err = runMySQLExec(fmt.Sprintf(
			"CREATE USER '%s'@'%s' IDENTIFIED BY '%s';", req.Username, req.Host, req.Password))
		if err != nil {
			fail(c, 500, "创建用户失败: "+err.Error())
			return
		}
		// Grant privileges
		if req.DBName != "" && req.DBName != "*" {
			err = runMySQLExec(fmt.Sprintf(
				"GRANT %s ON `%s`.* TO '%s'@'%s';", req.Privileges, req.DBName, req.Username, req.Host))
		} else {
			err = runMySQLExec(fmt.Sprintf(
				"GRANT %s ON *.* TO '%s'@'%s';", req.Privileges, req.Username, req.Host))
		}
		if err == nil {
			runMySQLExec("FLUSH PRIVILEGES;")
		}

	case "postgresql":
		err = runPSQLExec(fmt.Sprintf(
			"CREATE USER %s WITH PASSWORD '%s';", req.Username, req.Password))
		if err != nil {
			fail(c, 500, "创建用户失败: "+err.Error())
			return
		}
		if req.DBName != "" && req.DBName != "*" {
			err = runPSQLExec(fmt.Sprintf(
				"GRANT %s ON DATABASE \"%s\" TO %s;", req.Privileges, req.DBName, req.Username))
		} else if req.Privileges == "ALL" {
			err = runPSQLExec(fmt.Sprintf(
				"ALTER USER %s CREATEDB;", req.Username))
		}

	case "redis":
		err = exec.Command("redis-cli", "ACL", "SETUSER", req.Username,
			"on", ">"+req.Password, "+@all").Run()

	default:
		fail(c, 400, "不支持的数据库类型")
		return
	}

	if err != nil {
		fail(c, 500, "授权失败: "+err.Error())
		return
	}

	// Save to local DB
	user := model.DBUser{
		InstanceID: instance.ID,
		Username:   req.Username,
		Host:       req.Host,
		DBName:     req.DBName,
		Privileges: req.Privileges,
	}
	config.DB.Create(&user)

		ok(c, user)
}

// DeleteDBUser 删除数据库用户
// DELETE /api/databases/users/:did
func DeleteDBUser(c *gin.Context) {
	var user model.DBUser
	if err := config.DB.First(&user, c.Param("did")).Error; err != nil {
		fail(c, 404, "用户记录不存在")
		return
	}

	var instance model.DBInstance
	config.DB.First(&instance, user.InstanceID)

	var err error
	switch instance.Type {
	case "mysql":
		err = runMySQLExec(fmt.Sprintf(
			"DROP USER IF EXISTS '%s'@'%s';", user.Username, user.Host))
	case "postgresql":
		err = runPSQLExec(fmt.Sprintf(
			"DROP USER IF EXISTS %s;", user.Username))
	case "redis":
		err = exec.Command("redis-cli", "ACL", "DELUSER", user.Username).Run()
	default:
		fail(c, 400, "不支持的数据库类型")
		return
	}

	if err != nil {
		fail(c, 500, "删除用户失败: "+err.Error())
		return
	}

	config.DB.Delete(&user)
	okMsg(c, nil, "删除成功")
}

// ========== 备份管理 ==========

// ListDBBackups 列出备份
// GET /api/databases/instances/:id/backups
func ListDBBackups(c *gin.Context) {
	var backups []model.DBBackup
	config.DB.Where("instance_id = ?", c.Param("id")).Order("created_at desc").Find(&backups)
	ok(c, backups)
}

// CreateDBBackup 创建备份
// POST /api/databases/instances/:id/backups
func CreateDBBackup(c *gin.Context) {
	var instance model.DBInstance
	if err := config.DB.First(&instance, c.Param("id")).Error; err != nil {
		fail(c, 404, "实例不存在")
		return
	}

	var req struct {
		DBName string `json:"db_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, err.Error())
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	backupDir := "/tmp/opsmanage_backups"
	exec.Command("mkdir", "-p", backupDir).Run()

	var filePath string
	var err error

	switch instance.Type {
	case "mysql":
		filePath = fmt.Sprintf("%s/mysql_%s_%s.sql", backupDir, req.DBName, timestamp)
		var cmd *exec.Cmd
		if instance.RootPass != "" {
			cmd = exec.Command("mysqldump", "-uroot", "-p"+instance.RootPass,
				"--single-transaction", req.DBName)
		} else {
			cmd = exec.Command("mysqldump", "-uroot",
				"--single-transaction", req.DBName)
		}
		var out []byte
		out, err = cmd.Output()
		if err == nil {
			err = writeFile(filePath, string(out))
		}

	case "postgresql":
		filePath = fmt.Sprintf("%s/pg_%s_%s.sql", backupDir, req.DBName, timestamp)
		cmd := exec.Command("sudo", "-u", "postgres", "pg_dump", req.DBName)
		var out []byte
		out, err = cmd.Output()
		if err == nil {
			err = writeFile(filePath, string(out))
		}

	case "redis":
		filePath = fmt.Sprintf("%s/redis_%s.rdb", backupDir, timestamp)
		err = exec.Command("redis-cli", "BGSAVE").Run()

	default:
		fail(c, 400, "不支持的数据库类型")
		return
	}

	status := "success"
	if err != nil {
		status = "failed"
		filePath = ""
	}

	// Get file size
	size := int64(0)
	if filePath != "" {
		cmd := exec.Command("stat", "-c", "%s", filePath)
		if out, err := cmd.Output(); err == nil {
			fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &size)
		}
	}

	backup := model.DBBackup{
		InstanceID: instance.ID,
		DBName:     req.DBName,
		FilePath:   filePath,
		Size:       size,
		Status:     status,
	}
	config.DB.Create(&backup)

	if err != nil {
		fail(c, 500, "备份失败: "+err.Error())
		return
	}

	ok(c, backup)
}

// RestoreDBBackup 恢复备份
// POST /api/databases/backups/:bid/restore
func RestoreDBBackup(c *gin.Context) {
	var backup model.DBBackup
	if err := config.DB.First(&backup, c.Param("bid")).Error; err != nil {
		fail(c, 404, "备份不存在")
		return
	}

	if backup.Status != "success" || backup.FilePath == "" {
		fail(c, 400, "备份文件无效")
		return
	}

	var instance model.DBInstance
	config.DB.First(&instance, backup.InstanceID)

	var err error
	switch instance.Type {
	case "mysql":
		var cmd *exec.Cmd
		if instance.RootPass != "" {
			cmd = exec.Command("mysql", "-uroot", "-p"+instance.RootPass, backup.DBName)
		} else {
			cmd = exec.Command("mysql", "-uroot", backup.DBName)
		}
		// Pipe file to mysql
		cmd.Args = append(cmd.Args, "<", backup.FilePath)
		err = exec.Command("sh", "-c",
			fmt.Sprintf("mysql -uroot %s < %s", backup.DBName, backup.FilePath)).Run()

	case "postgresql":
		err = exec.Command("sudo", "-u", "postgres", "psql", "-d", backup.DBName,
			"-f", backup.FilePath).Run()

	default:
		fail(c, 400, "不支持的数据库类型")
		return
	}

	if err != nil {
		fail(c, 500, "恢复失败: "+err.Error())
		return
	}

	okMsg(c, nil, "恢复成功")
}

// ========== 辅助函数 ==========

// ========== 兼容函数（installer.go 使用） ==========

func getMySQLVersion() string    { return detectDBVersion("mysql") }
func getPostgreSQLVersion() string { return detectDBVersion("postgresql") }
func getRedisVersion() string     { return detectDBVersion("redis") }
func isMySQLInstalled() bool      { return detectDBVersion("mysql") != "" }
func isPostgreSQLInstalled() bool  { return detectDBVersion("postgresql") != "" }
func isRedisInstalled() bool       { return detectDBVersion("redis") != "" }

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func writeFile(path, content string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("cat > %s", shellQuote(path)))
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}
