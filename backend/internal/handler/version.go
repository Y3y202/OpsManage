package handler

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ========== 服务版本管理 ==========

// ServiceVersionInfo 单个服务的版本信息
type ServiceVersionInfo struct {
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	CurrentVersion  string   `json:"current_version"`
	Installed       bool     `json:"installed"`
	Running         bool     `json:"running"`
	AvailableVersions []string `json:"available_versions"`
	InstalledVersions []string `json:"installed_versions"`
}

// GetServiceVersions 获取所有服务的版本信息
func GetServiceVersions(c *gin.Context) {
	services := []ServiceVersionInfo{
		getNginxVersionInfo(),
		getMySQLVersionInfo(),
		getPostgreSQLVersionInfo(),
		getRedisVersionInfo(),
	}
	success(c, services)
}

// GetServiceVersionsByType 获取指定服务的版本信息
func GetServiceVersionsByType(c *gin.Context) {
	serviceType := c.Param("type")
	var info ServiceVersionInfo

	switch strings.ToLower(serviceType) {
	case "nginx":
		info = getNginxVersionInfo()
	case "mysql":
		info = getMySQLVersionInfo()
	case "postgresql", "postgres":
		info = getPostgreSQLVersionInfo()
	case "redis":
		info = getRedisVersionInfo()
	default:
		fail(c, 400, "不支持的服务类型: "+serviceType)
		return
	}
	success(c, info)
}

// SwitchServiceVersion 切换服务版本
func SwitchServiceVersion(c *gin.Context) {
	serviceType := c.Param("type")
	var req struct {
		Version string `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: 需要指定目标版本")
		return
	}

	// 检查是否是当前版本
	var current string
	switch strings.ToLower(serviceType) {
	case "nginx":
		current = getNginxVersion()
	case "mysql":
		current = getMySQLVersion()
	case "postgresql", "postgres":
		current = getPostgreSQLVersion()
	case "redis":
		current = getRedisVersion()
	default:
		fail(c, 400, "不支持的服务类型: "+serviceType)
		return
	}

	// 提取主版本号比较
	if extractMajorVersion(current) == extractMajorVersion(req.Version) {
		fail(c, 400, "当前已是该版本: "+current)
		return
	}

	// 异步执行版本切换
	taskID := fmt.Sprintf("switch-%s-%d", serviceType, time.Now().Unix())
	createProgress(taskID, serviceType)

	go runVersionSwitch(taskID, serviceType, req.Version)

	success(c, gin.H{"task_id": taskID, "message": fmt.Sprintf("正在切换 %s 到版本 %s ...", serviceType, req.Version)})
}

// ========== Nginx 版本信息 ==========

func getNginxVersionInfo() ServiceVersionInfo {
	info := ServiceVersionInfo{
		Type:      "nginx",
		Name:      "Nginx",
		Installed: isNginxInstalled(),
		Running:   isNginxRunning(),
	}
	if info.Installed {
		info.CurrentVersion = extractMajorVersion(getNginxVersion())
		if info.CurrentVersion == "" {
			info.CurrentVersion = getNginxVersion()
		}
	}

	// 从 nginx.org 仓库获取可用版本
	info.AvailableVersions = getAptAvailableVersions("nginx")
	info.InstalledVersions = getAptInstalledVersions("nginx")

	return info
}

// ========== MySQL 版本信息 ==========

func getMySQLVersionInfo() ServiceVersionInfo {
	info := ServiceVersionInfo{
		Type:      "mysql",
		Name:      "MySQL",
		Installed: isMySQLInstalled(),
		Running:   isMySQLRunning(),
	}
	if info.Installed {
		info.CurrentVersion = extractMajorVersion(getMySQLVersion())
		if info.CurrentVersion == "" {
			info.CurrentVersion = getMySQLVersion()
		}
	}

	info.AvailableVersions = getMySQLAvailableVersions()
	info.InstalledVersions = getAptInstalledVersions("mysql-server")

	// 也检查 MariaDB
	mariaVersions := getAptInstalledVersions("mariadb-server")
	if len(mariaVersions) > 0 {
		info.InstalledVersions = append(info.InstalledVersions, mariaVersions...)
	}

	return info
}

// getMySQLAvailableVersions 获取 MySQL 可用版本（含 5.7）
func getMySQLAvailableVersions() []string {
	// 从 mysql-server 和 mysql-community-server 获取
	versionMap := make(map[string]bool)

	for _, pkg := range []string{"mysql-server", "mysql-community-server"} {
		cmd := exec.Command("apt-cache", "madison", pkg)
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				version := strings.TrimSpace(parts[1])
				majorVer := extractMajorVersion(version)
				if majorVer != "" && isValidVersion(majorVer) {
					versionMap[majorVer] = true
				}
			}
		}
	}

	versions := make([]string, 0, len(versionMap))
	for v := range versionMap {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return versions
}

// ========== PostgreSQL 版本信息 ==========

func getPostgreSQLVersionInfo() ServiceVersionInfo {
	info := ServiceVersionInfo{
		Type:      "postgresql",
		Name:      "PostgreSQL",
		Installed: isPostgreSQLInstalled(),
		Running:   isPostgreSQLRunning(),
	}
	if info.Installed {
		info.CurrentVersion = getPostgreSQLVersion()
	}

	// PostgreSQL 特殊：查看 postgresql-XX 包来获取可用版本
	info.AvailableVersions = getPostgreSQLAvailableVersions()
	info.InstalledVersions = getPostgreSQLClusters()

	return info
}

// getPostgreSQLAvailableVersions 获取 PostgreSQL 可用版本
func getPostgreSQLAvailableVersions() []string {
	cmd := exec.Command("apt-cache", "search", "^postgresql-[0-9]")
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	versionMap := make(map[string]bool)
	re := regexp.MustCompile(`^postgresql-(\d+)\s`)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			versionMap[matches[1]] = true
		}
	}

	versions := make([]string, 0, len(versionMap))
	for v := range versionMap {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return versions
}

// ========== Redis 版本信息 ==========

func getRedisVersionInfo() ServiceVersionInfo {
	info := ServiceVersionInfo{
		Type:      "redis",
		Name:      "Redis",
		Installed: isRedisInstalled(),
		Running:   isRedisRunning(),
	}
	if info.Installed {
		info.CurrentVersion = extractMajorVersion(getRedisVersion())
		if info.CurrentVersion == "" {
			info.CurrentVersion = getRedisVersion()
		}
	}

	info.AvailableVersions = getAptAvailableVersions("redis-server")
	info.InstalledVersions = getAptInstalledVersions("redis-server")

	return info
}

// ========== 辅助函数 ==========

// isMySQLRunning 检查 MySQL 是否运行
func isMySQLRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "mysql")
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		return true
	}
	// 也检查 mariadb
	cmd = exec.Command("systemctl", "is-active", "mariadb")
	output, err = cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) == "active"
}

// ========== PostgreSQL 版本信息 ==========

// isPostgreSQLRunning 检查 PostgreSQL 是否运行
func isPostgreSQLRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "postgresql")
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		return true
	}
	// 检查 pg_lsclusters
	cmd = exec.Command("pg_lsclusters", "-h")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 4 && fields[3] == "online" {
				return true
			}
		}
	}
	return false
}

// ========== Redis 版本信息 ==========

// isRedisRunning 检查 Redis 是否运行
func isRedisRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "redis-server")
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) == "active" {
		return true
	}
	cmd = exec.Command("systemctl", "is-active", "redis")
	output, err = cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) == "active"
}

// getAptAvailableVersions 从 apt 获取可用版本列表
func getAptAvailableVersions(packageName string) []string {
	cmd := exec.Command("apt-cache", "madison", packageName)
	output, err := cmd.Output()
	if err != nil {
		return getAptPolicyVersions(packageName)
	}

	versionMap := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 格式: nginx | 1.30.0-1~noble | http://nginx.org/packages/ubuntu noble/nginx amd64 Packages
		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			version := strings.TrimSpace(parts[1])
			// 提取纯版本号
			majorVer := extractMajorVersion(version)
			if majorVer != "" && isValidVersion(majorVer) {
				versionMap[majorVer] = true
			}
		}
	}

	versions := make([]string, 0, len(versionMap))
	for v := range versionMap {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return versions
}

// isValidVersion 检查是否是有效版本号（排除构建号等）
func isValidVersion(version string) bool {
	// 版本号应该是类似 "1.30", "8.0", "7.0" 这样的格式
	re := regexp.MustCompile(`^\d+\.\d+$`)
	return re.MatchString(version)
}

// getAptPolicyVersions 从 apt-cache policy 获取版本
func getAptPolicyVersions(packageName string) []string {
	cmd := exec.Command("apt-cache", "policy", packageName)
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	versionMap := make(map[string]bool)
	re := regexp.MustCompile(`(\d+\.\d+\.\d+[-\w.]*)`)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Candidate") || strings.Contains(line, "Installed") {
			continue
		}
		matches := re.FindAllString(line, -1)
		for _, m := range matches {
			majorVer := extractMajorVersion(m)
			if majorVer != "" {
				versionMap[majorVer] = true
			}
		}
	}

	versions := make([]string, 0, len(versionMap))
	for v := range versionMap {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return versions
}

// getAptInstalledVersions 获取已安装版本
func getAptInstalledVersions(packageName string) []string {
	cmd := exec.Command("dpkg", "-l", packageName)
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	versions := []string{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ii") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				ver := extractMajorVersion(fields[2])
				if ver != "" {
					versions = append(versions, ver)
				}
			}
		}
	}
	return versions
}

// getPostgreSQLClusters 获取 PostgreSQL 已安装的集群版本
func getPostgreSQLClusters() []string {
	cmd := exec.Command("pg_lsclusters", "-h")
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	versionMap := make(map[string]bool)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			versionMap[fields[0]] = true
		}
	}

	versions := make([]string, 0, len(versionMap))
	for v := range versionMap {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return versions
}

// extractMajorVersion 从版本字符串提取主版本号
// 例如: "1.30.0-1~noble" -> "1.30", "8.0.45-0ubuntu0.24.04.1" -> "8.0"
func extractMajorVersion(version string) string {
	if version == "" {
		return ""
	}
	re := regexp.MustCompile(`(\d+\.\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) > 1 {
		return matches[1]
	}
	return version
}

// ========== 版本切换执行 ==========

func runVersionSwitch(taskID, serviceType, targetVersion string) {
	setProgress(taskID, "running", 5, fmt.Sprintf("开始切换 %s 到版本 %s ...", serviceType, targetVersion))

	var err error
	switch strings.ToLower(serviceType) {
	case "nginx":
		err = switchNginxVersion(taskID, targetVersion)
	case "mysql":
		err = switchMySQLVersion(taskID, targetVersion)
	case "postgresql", "postgres":
		err = switchPostgreSQLVersion(taskID, targetVersion)
	case "redis":
		err = switchRedisVersion(taskID, targetVersion)
	default:
		setProgress(taskID, "failed", 0, "不支持的服务类型: "+serviceType)
		return
	}

	if err != nil {
		setProgress(taskID, "failed", 100, "切换失败: "+err.Error())
		addLog("error", "version", fmt.Sprintf("%s 版本切换失败: %v", serviceType, err))
	} else {
		setProgress(taskID, "success", 100, fmt.Sprintf("%s 版本切换成功！", serviceType))
		addLog("info", "version", fmt.Sprintf("%s 版本已切换到 %s", serviceType, targetVersion))
		updateDBInstance(serviceType)
	}
}

// switchNginxVersion 切换 Nginx 版本
func switchNginxVersion(taskID, targetVersion string) error {
	currentVersion := getNginxVersion()
	if extractMajorVersion(currentVersion) == targetVersion {
		return fmt.Errorf("当前已是版本 %s", currentVersion)
	}

	addInstallLog(taskID, fmt.Sprintf("当前版本: %s, 目标版本: %s", currentVersion, targetVersion))
	setProgress(taskID, "running", 10, "停止 Nginx 服务...")

	// 停止服务
	exec.Command("systemctl", "stop", "nginx").Run()

	setProgress(taskID, "running", 20, "查找目标版本包...")

	// 查找完整版本号
	fullVersion := findFullPackageVersion("nginx", targetVersion)
	if fullVersion == "" {
		// 尝试安装指定版本
		fullVersion = targetVersion
	}

	addInstallLog(taskID, fmt.Sprintf("安装版本: %s", fullVersion))
	setProgress(taskID, "running", 30, "安装目标版本...")

	// 安装指定版本
	cmd := exec.Command("apt-get", "install", "-y", "--allow-downgrades",
		fmt.Sprintf("nginx=%s", fullVersion))
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	output, err := cmd.CombinedOutput()
	if err != nil {
		addInstallLog(taskID, fmt.Sprintf("安装失败: %s", string(output)))
		// 尝试重新启动旧版本
		exec.Command("systemctl", "start", "nginx").Run()
		return fmt.Errorf("安装失败: %s", string(output))
	}

	setProgress(taskID, "running", 70, "验证安装...")

	// 验证新版本
	newVersion := getNginxVersion()
	addInstallLog(taskID, fmt.Sprintf("新版本: %s", newVersion))

	setProgress(taskID, "running", 80, "启动 Nginx 服务...")

	// 测试配置并启动
	testCmd := exec.Command("nginx", "-t")
	if err := testCmd.Run(); err != nil {
		addInstallLog(taskID, "⚠️ 配置测试失败，尝试自动修复...")
		// 基本修复：确保 sites-enabled 目录存在
		exec.Command("mkdir", "-p", "/etc/nginx/sites-enabled").Run()
	}

	exec.Command("systemctl", "start", "nginx").Run()

	setProgress(taskID, "running", 95, "完成！")
	addInstallLog(taskID, fmt.Sprintf("✅ Nginx 已切换到 %s", newVersion))
	return nil
}

// switchMySQLVersion 切换 MySQL 版本
func switchMySQLVersion(taskID, targetVersion string) error {
	currentVersion := getMySQLVersion()
	currentMajor := extractMajorVersion(currentVersion)
	if currentMajor == targetVersion {
		return fmt.Errorf("当前已是版本 %s", currentVersion)
	}

	addInstallLog(taskID, fmt.Sprintf("当前版本: %s, 目标版本: %s", currentVersion, targetVersion))
	setProgress(taskID, "running", 10, "备份数据库...")

	// 备份所有数据库
	backupPath := fmt.Sprintf("/tmp/mysql_backup_%s.sql", time.Now().Format("20060102150405"))
	cmd := exec.Command("mysqldump", "-u", "root", "--all-databases", "--single-transaction", "-r", backupPath)
	cmd.Run()
	addInstallLog(taskID, fmt.Sprintf("备份完成: %s", backupPath))

	setProgress(taskID, "running", 25, "停止并移除当前 MySQL...")
	exec.Command("systemctl", "stop", "mysql").Run()
	exec.Command("systemctl", "stop", "mysqld").Run()

	// 完全移除当前版本
	purgeCmd := exec.Command("apt-get", "purge", "-y", "mysql-*", "mysql*", "mariadb-*")
	purgeCmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	purgeCmd.Run()
	exec.Command("apt-get", "autoremove", "-y").Run()

	setProgress(taskID, "running", 40, fmt.Sprintf("安装 MySQL %s ...", targetVersion))

	var installCmd *exec.Cmd
	if targetVersion == "5.7" {
		// MySQL 5.7 需要特殊处理：从 bionic 仓库安装
		addInstallLog(taskID, "处理 Ubuntu 24.04 兼容性...")

		// 1. 安装 libtinfo5（真正的 Ubuntu 18.04 包）
		addInstallLog(taskID, "安装 libtinfo5 兼容库...")
		libtinfoPath := "/usr/lib/x86_64-linux-gnu/libtinfo.so.5"
		if _, err := os.Stat(libtinfoPath); os.IsNotExist(err) {
			// 下载并安装真正的 libtinfo5
			exec.Command("wget", "-q",
				"http://archive.ubuntu.com/ubuntu/pool/main/n/ncurses/libtinfo5_6.1-1ubuntu1_amd64.deb",
				"-O", "/tmp/libtinfo5_real.deb").Run()
			exec.Command("dpkg", "-i", "--force-depends", "/tmp/libtinfo5_real.deb").Run()
		}
		// 如果还是没有，创建符号链接
		if _, err := os.Stat(libtinfoPath); os.IsNotExist(err) {
			exec.Command("ln", "-sf",
				"/usr/lib/x86_64-linux-gnu/libtinfo.so.6",
				libtinfoPath).Run()
		}

		// 2. 安装 libaio1 兼容包
		libaioPath := "/usr/lib/x86_64-linux-gnu/libaio.so.1"
		if _, err := os.Stat(libaioPath); os.IsNotExist(err) {
			addInstallLog(taskID, "创建 libaio1 兼容链接...")
			exec.Command("ln", "-sf",
				"/usr/lib/x86_64-linux-gnu/libaio.so.1t64",
				libaioPath).Run()
			exec.Command("ldconfig").Run()
		}

		fullVersion := findFullVersionFromMadison("mysql-community-server", targetVersion)
		if fullVersion == "" {
			fullVersion = "5.7.42-1ubuntu18.04"
		}
		clientVersion := findFullVersionFromMadison("mysql-community-client", targetVersion)
		if clientVersion == "" {
			clientVersion = fullVersion
		}

		addInstallLog(taskID, fmt.Sprintf("安装 mysql-community-client/server %s", fullVersion))

		installCmd = exec.Command("apt-get", "install", "-y", "--allow-downgrades",
			fmt.Sprintf("mysql-community-client=%s", clientVersion),
			fmt.Sprintf("mysql-community-server=%s", fullVersion),
			"--fix-broken")
	} else {
		// MySQL 8.0 / 9.6 从 Ubuntu 默认仓库或 MySQL 官方仓库安装
		addInstallLog(taskID, "安装 mysql-server ...")
		installCmd = exec.Command("apt-get", "install", "-y", "--allow-downgrades", "mysql-server")
	}

	installCmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	output, err := installCmd.CombinedOutput()
	if err != nil {
		addInstallLog(taskID, fmt.Sprintf("安装失败: %s", string(output)))
		// 尝试恢复安装 8.0
		restoreCmd := exec.Command("apt-get", "install", "-y", "mysql-server")
		restoreCmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		restoreCmd.Run()
		return fmt.Errorf("安装失败: %s", string(output))
	}

	setProgress(taskID, "running", 75, "启动 MySQL 服务...")
	exec.Command("systemctl", "daemon-reload").Run()

	// 检查数据目录兼容性，不兼容时重新初始化
	dataDir := "/var/lib/mysql"
	if _, err := os.Stat(dataDir + "/ibdata1"); err == nil {
		// 尝试启动，失败则重新初始化
		if exec.Command("systemctl", "start", "mysql").Run() != nil {
			addInstallLog(taskID, "数据目录不兼容，重新初始化...")
			exec.Command("systemctl", "stop", "mysql").Run()
			os.RemoveAll(dataDir)
			os.MkdirAll(dataDir, 0755)
			exec.Command("chown", "mysql:mysql", dataDir).Run()
			exec.Command("mysqld", "--initialize-insecure", "--user=mysql").Run()
			exec.Command("systemctl", "start", "mysql").Run()
			// 设置默认密码
			time.Sleep(3 * time.Second)
			exec.Command("mysql", "-u", "root",
				"-e", "ALTER USER 'root'@'localhost' IDENTIFIED BY 'admin123'; FLUSH PRIVILEGES;").Run()
			addInstallLog(taskID, "数据库已初始化，默认密码: admin123")
		}
	} else {
		// 数据目录不存在，初始化
		addInstallLog(taskID, "初始化 MySQL 数据目录...")
		os.MkdirAll(dataDir, 0755)
		exec.Command("chown", "mysql:mysql", dataDir).Run()
		exec.Command("mysqld", "--initialize-insecure", "--user=mysql").Run()
		exec.Command("systemctl", "start", "mysql").Run()
		time.Sleep(3 * time.Second)
		exec.Command("mysql", "-u", "root",
			"-e", "ALTER USER 'root'@'localhost' IDENTIFIED BY 'admin123'; FLUSH PRIVILEGES;").Run()
		addInstallLog(taskID, "数据库已初始化，默认密码: admin123")
	}

	setProgress(taskID, "running", 90, "验证版本...")
	newVersion := getMySQLVersion()
	addInstallLog(taskID, fmt.Sprintf("✅ MySQL 已切换到 %s", newVersion))

	setProgress(taskID, "running", 95, "完成！")
	return nil
}

// switchPostgreSQLVersion 切换 PostgreSQL 版本
func switchPostgreSQLVersion(taskID, targetVersion string) error {
	currentVersion := getPostgreSQLVersion()
	currentMajor := extractMajorVersion(currentVersion)
	if currentMajor == targetVersion {
		return fmt.Errorf("当前已是版本 %s", currentVersion)
	}

	addInstallLog(taskID, fmt.Sprintf("当前版本: %s, 目标版本: %s", currentVersion, targetVersion))

	// PostgreSQL 特殊处理：可以同时安装多个版本，通过 pg_ctlcluster 切换
	clusters := getPostgreSQLClusters()
	targetInstalled := false
	for _, v := range clusters {
		if v == targetVersion {
			targetInstalled = true
			break
		}
	}

	if !targetInstalled {
		setProgress(taskID, "running", 15, fmt.Sprintf("安装 PostgreSQL %s ...", targetVersion))

		// 安装目标版本
		pkg := fmt.Sprintf("postgresql-%s", targetVersion)
		cmd := exec.Command("apt-get", "install", "-y", pkg)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		output, err := cmd.CombinedOutput()
		if err != nil {
			addInstallLog(taskID, fmt.Sprintf("安装失败: %s", string(output)))
			return fmt.Errorf("安装 PostgreSQL %s 失败: %s", targetVersion, string(output))
		}
		addInstallLog(taskID, fmt.Sprintf("PostgreSQL %s 安装成功", targetVersion))
	}

	setProgress(taskID, "running", 50, "停止旧版本集群...")

	// 停止旧版本集群
	if currentMajor != "" {
		exec.Command("pg_ctlcluster", currentMajor, "main", "stop").Run()
	}

	setProgress(taskID, "running", 65, "启动目标版本集群...")

	// 启动目标版本集群
	cmd := exec.Command("pg_ctlcluster", targetVersion, "main", "start")
	output, err := cmd.CombinedOutput()
	if err != nil {
		addInstallLog(taskID, fmt.Sprintf("启动失败: %s", string(output)))
		return fmt.Errorf("启动 PostgreSQL %s 失败: %s", targetVersion, string(output))
	}

	setProgress(taskID, "running", 80, "验证版本...")
	newVersion := getPostgreSQLVersion()
	addInstallLog(taskID, fmt.Sprintf("✅ PostgreSQL 已切换到 %s", newVersion))

	setProgress(taskID, "running", 95, "完成！")
	return nil
}

// switchRedisVersion 切换 Redis 版本
func switchRedisVersion(taskID, targetVersion string) error {
	currentVersion := getRedisVersion()
	if extractMajorVersion(currentVersion) == targetVersion {
		return fmt.Errorf("当前已是版本 %s", currentVersion)
	}

	addInstallLog(taskID, fmt.Sprintf("当前版本: %s, 目标版本: %s", currentVersion, targetVersion))
	setProgress(taskID, "running", 10, "备份 Redis 数据...")

	// 备份 RDB
	exec.Command("redis-cli", "BGSAVE").Run()
	time.Sleep(2 * time.Second)

	setProgress(taskID, "running", 25, "停止 Redis 服务...")
	exec.Command("systemctl", "stop", "redis-server").Run()
	exec.Command("systemctl", "stop", "redis").Run()

	setProgress(taskID, "running", 35, "安装目标版本...")

	fullVersion := findFullPackageVersion("redis-server", targetVersion)
	if fullVersion == "" {
		fullVersion = targetVersion
	}

	addInstallLog(taskID, fmt.Sprintf("安装版本: %s", fullVersion))

	cmd := exec.Command("apt-get", "install", "-y", "--allow-downgrades",
		fmt.Sprintf("redis-server=%s", fullVersion))
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	output, err := cmd.CombinedOutput()
	if err != nil {
		addInstallLog(taskID, fmt.Sprintf("安装失败: %s", string(output)))
		exec.Command("systemctl", "start", "redis-server").Run()
		return fmt.Errorf("安装失败: %s", string(output))
	}

	setProgress(taskID, "running", 75, "启动 Redis 服务...")
	exec.Command("systemctl", "start", "redis-server").Run()

	setProgress(taskID, "running", 90, "验证版本...")
	newVersion := getRedisVersion()
	addInstallLog(taskID, fmt.Sprintf("✅ Redis 已切换到 %s", newVersion))

	setProgress(taskID, "running", 95, "完成！")
	return nil
}

// findFullPackageVersion 查找包的完整版本号
func findFullPackageVersion(packageName, majorVersion string) string {
	// 先查指定包名
	result := findFullVersionFromMadison(packageName, majorVersion)
	if result != "" {
		return result
	}
	// 对于 MySQL，也查 mysql-community-server
	if packageName == "mysql-server" {
		return findFullVersionFromMadison("mysql-community-server", majorVersion)
	}
	return ""
}

func findFullVersionFromMadison(packageName, majorVersion string) string {
	cmd := exec.Command("apt-cache", "madison", packageName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, majorVersion) {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
