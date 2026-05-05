package handler

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
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
	"github.com/gorilla/websocket"
)

// ========== Nginx 服务管理 ==========

// NginxStatus Nginx 服务状态
func NginxStatus(c *gin.Context) {
	info := map[string]interface{}{
		"installed":   isNginxInstalled(),
		"running":     isNginxRunning(),
		"version":     getNginxVersion(),
		"config_path": "/etc/nginx/nginx.conf",
		"conf_d":      "/etc/nginx/conf.d",
		"sites_path":  "/etc/nginx/sites-enabled",
	}
	var siteCount int64
	config.DB.Model(&model.NginxSite{}).Count(&siteCount)
	info["site_count"] = siteCount
	success(c, info)
}

// NginxService 操作 Nginx 服务
func NginxService(c *gin.Context) {
	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	if !isNginxInstalled() {
		fail(c, 400, "Nginx 未安装")
		return
	}

	var cmd *exec.Cmd
	switch req.Action {
	case "start":
		cmd = exec.Command("systemctl", "start", "nginx")
	case "stop":
		cmd = exec.Command("systemctl", "stop", "nginx")
	case "restart":
		cmd = exec.Command("systemctl", "restart", "nginx")
	case "reload":
		cmd = exec.Command("nginx", "-t")
		if err := cmd.Run(); err != nil {
			fail(c, 400, "Nginx 配置测试失败，请检查配置文件")
			return
		}
		cmd = exec.Command("nginx", "-s", "reload")
	default:
		fail(c, 400, "不支持的操作")
		return
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fail(c, 500, "操作失败: "+string(output))
		return
	}
	addLog("info", "nginx", "Nginx "+req.Action+" 操作成功")
	success(c, gin.H{"status": isNginxRunning(), "message": "操作成功"})
}

// NginxTestConfig 测试 Nginx 配置
func NginxTestConfig(c *gin.Context) {
	cmd := exec.Command("nginx", "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "配置测试失败", "data": gin.H{"valid": false, "output": string(output)}})
		return
	}
	success(c, gin.H{"valid": true, "output": string(output)})
}

// ========== Nginx 站点管理 ==========

// ListNginxSites 获取 Nginx 站点列表
func ListNginxSites(c *gin.Context) {
	var sites []model.NginxSite
	var total int64
	query := config.DB.Model(&model.NginxSite{})
	query.Count(&total)
	paginate(c, query).Order("id desc").Find(&sites)
	pageResult(c, sites, total)
}

// CreateNginxSiteReq 创建站点请求
type CreateNginxSiteReq struct {
	Name       string `json:"name" binding:"required"`
	Domain     string `json:"domain" binding:"required"`
	Root       string `json:"root" binding:"required"`
	Port       int    `json:"port"`
	ProxyType  string `json:"proxy_type"`
	ProxyPass  string `json:"proxy_pass"`
	Gzip       *bool  `json:"gzip"`
	Remark     string `json:"remark"`
}

// CreateNginxSite 创建 Nginx 站点
func CreateNginxSite(c *gin.Context) {
	var req CreateNginxSiteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	if req.Port == 0 {
		req.Port = 80
	}
	if req.ProxyType == "" {
		req.ProxyType = "static"
	}

	os.MkdirAll(req.Root, 0755)

	gzip := true
	if req.Gzip != nil {
		gzip = *req.Gzip
	}

	site := model.NginxSite{
		Name:       req.Name,
		Domain:     req.Domain,
		Root:       req.Root,
		Port:       req.Port,
		ProxyType:  req.ProxyType,
		ProxyPass:  req.ProxyPass,
		Gzip:       gzip,
		Status:     "running",
		Remark:     req.Remark,
	}

	if err := config.DB.Create(&site).Error; err != nil {
		fail(c, 500, "创建失败")
		return
	}

	confPath := generateSiteNginxConf(&site)
	site.ConfigFile = confPath
	content, _ := os.ReadFile(confPath)
	site.ConfigContent = string(content)
	config.DB.Model(&site).Updates(map[string]interface{}{
		"config_file":    confPath,
		"config_content": site.ConfigContent,
	})

	testCmd := exec.Command("nginx", "-t")
	if err := testCmd.Run(); err != nil {
		os.Remove(confPath)
		config.DB.Delete(&site)
		fail(c, 400, "Nginx 配置测试失败，请检查域名和路径设置")
		return
	}
	reloadNginx()

	addLog("info", "nginx", "创建站点: "+site.Name+" ("+site.Domain+")")
	success(c, site)
}

// GetNginxSite 获取站点详情
func GetNginxSite(c *gin.Context) {
	id := c.Param("id")
	var site model.NginxSite
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	if site.ConfigFile != "" {
		content, err := os.ReadFile(site.ConfigFile)
		if err == nil {
			site.ConfigContent = string(content)
		}
	}
	success(c, site)
}

// UpdateNginxSite 更新站点
func UpdateNginxSite(c *gin.Context) {
	id := c.Param("id")
	var site model.NginxSite
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	config.DB.Model(&site).Updates(req)
	config.DB.First(&site, id)

	confPath := generateSiteNginxConf(&site)
	content, _ := os.ReadFile(confPath)
	config.DB.Model(&site).Updates(map[string]interface{}{
		"config_file":    confPath,
		"config_content": string(content),
	})

	testNginxAndReload()
	success(c, site)
}

// DeleteNginxSite 删除站点
func DeleteNginxSite(c *gin.Context) {
	id := c.Param("id")
	var site model.NginxSite
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	if site.ConfigFile != "" {
		os.Remove(site.ConfigFile)
	}
	config.DB.Delete(&site)
	reloadNginx()
	addLog("info", "nginx", "删除站点: "+site.Name+" ("+site.Domain+")")
	success(c, nil)
}

// NginxSiteAction 站点启停
func NginxSiteAction(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")
	var site model.NginxSite
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	switch action {
	case "enable":
		config.DB.Model(&site).Update("status", "running")
		generateSiteNginxConf(&site)
	case "disable":
		config.DB.Model(&site).Update("status", "stopped")
		if site.ConfigFile != "" {
			os.Remove(site.ConfigFile)
		}
	default:
		fail(c, 400, "不支持的操作")
		return
	}
	reloadNginx()
	success(c, gin.H{"status": action})
}

// NginxSiteSSL 申请/管理 SSL 证书
func NginxSiteSSL(c *gin.Context) {
	id := c.Param("id")
	var site model.NginxSite
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var req struct {
		Action   string `json:"action" binding:"required"`
		Email    string `json:"email"`
		CertPath string `json:"cert_path"`
		KeyPath  string `json:"key_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	switch req.Action {
	case "enable":
		if req.CertPath != "" && req.KeyPath != "" {
			config.DB.Model(&site).Updates(map[string]interface{}{
				"ssl":      true,
				"ssl_cert": req.CertPath,
				"ssl_key":  req.KeyPath,
			})
		} else {
			go func() {
				acmeInstallAndCert(&site, req.Email)
			}()
			success(c, gin.H{"message": "正在申请 SSL 证书..."})
			return
		}
	case "disable":
		config.DB.Model(&site).Updates(map[string]interface{}{
			"ssl":      false,
			"ssl_cert": "",
			"ssl_key":  "",
		})
	case "renew":
		go func() {
			acmeRenewCert(&site)
		}()
		success(c, gin.H{"message": "正在续签 SSL 证书..."})
		return
	default:
		fail(c, 400, "不支持的操作")
		return
	}

	config.DB.First(&site, id)
	confPath := generateSiteNginxConf(&site)
	content, _ := os.ReadFile(confPath)
	config.DB.Model(&site).Updates(map[string]interface{}{
		"config_file":    confPath,
		"config_content": string(content),
	})
	reloadNginx()
	success(c, site)
}

// NginxSiteConfig 编辑站点配置文件
func NginxSiteConfig(c *gin.Context) {
	id := c.Param("id")
	var site model.NginxSite
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	if c.Request.Method == "GET" {
		if site.ConfigFile != "" {
			content, err := os.ReadFile(site.ConfigFile)
			if err == nil {
				site.ConfigContent = string(content)
			}
		}
		success(c, gin.H{"config_file": site.ConfigFile, "content": site.ConfigContent})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	if site.ConfigFile == "" {
		site.ConfigFile = fmt.Sprintf("/etc/nginx/conf.d/%s.conf", site.Domain)
	}

	backupPath := site.ConfigFile + ".bak." + time.Now().Format("20060102150405")
	if _, err := os.Stat(site.ConfigFile); err == nil {
		data, _ := os.ReadFile(site.ConfigFile)
		os.WriteFile(backupPath, data, 0644)
	}

	if err := os.WriteFile(site.ConfigFile, []byte(req.Content), 0644); err != nil {
		fail(c, 500, "写入配置文件失败")
		return
	}

	cmd := exec.Command("nginx", "-t")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if _, err := os.Stat(backupPath); err == nil {
			data, _ := os.ReadFile(backupPath)
			os.WriteFile(site.ConfigFile, data, 0644)
		}
		fail(c, 400, "配置测试失败，已回滚: "+string(output))
		return
	}

	config.DB.Model(&site).Update("config_content", req.Content)
	reloadNginx()
	addLog("info", "nginx", "编辑站点配置: "+site.Domain)
	success(c, gin.H{"message": "配置已保存并重载"})
}

// NginxSiteLogs 查看站点日志
func NginxSiteLogs(c *gin.Context) {
	id := c.Param("id")
	logType := c.DefaultQuery("type", "access")
	linesStr := c.DefaultQuery("lines", "100")
	lines, _ := strconv.Atoi(linesStr)

	var site model.NginxSite
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var logPath string
	if logType == "error" {
		logPath = fmt.Sprintf("/var/log/nginx/%s_error.log", site.Domain)
	} else {
		logPath = fmt.Sprintf("/var/log/nginx/%s_access.log", site.Domain)
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		success(c, gin.H{"logs": "", "path": logPath})
		return
	}

	cmd := exec.Command("tail", "-n", strconv.Itoa(lines), logPath)
	output, err := cmd.Output()
	if err != nil {
		fail(c, 500, "读取日志失败")
		return
	}
	success(c, gin.H{"logs": string(output), "path": logPath})
}

// NginxSiteLogStream WebSocket 实时日志
func NginxSiteLogStream(c *gin.Context) {
	id := c.Param("id")
	logType := c.DefaultQuery("type", "access")

	var site model.NginxSite
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var logPath string
	if logType == "error" {
		logPath = fmt.Sprintf("/var/log/nginx/%s_error.log", site.Domain)
	} else {
		logPath = fmt.Sprintf("/var/log/nginx/%s_access.log", site.Domain)
	}

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	cmd := exec.Command("tail", "-f", "-n", "50", logPath)
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			ws.WriteMessage(websocket.TextMessage, []byte(scanner.Text()))
		}
	}()

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			cmd.Process.Kill()
			return
		}
	}
}

// NginxStatusOverview Nginx 状态概览
func NginxStatusOverview(c *gin.Context) {
	overview := map[string]interface{}{
		"installed":     isNginxInstalled(),
		"running":       isNginxRunning(),
		"version":       getNginxVersion(),
		"active_conns":  getNginxActiveConns(),
	}
	var siteCount, runningCount int64
	config.DB.Model(&model.NginxSite{}).Count(&siteCount)
	config.DB.Model(&model.NginxSite{}).Where("status = ?", "running").Count(&runningCount)
	overview["total_sites"] = siteCount
	overview["running_sites"] = runningCount

	var sslCount int64
	config.DB.Model(&model.NginxSite{}).Where("ssl = ?", true).Count(&sslCount)
	overview["ssl_sites"] = sslCount

	success(c, overview)
}

// ========== 辅助函数 ==========

func isNginxInstalled() bool {
	_, err := exec.LookPath("nginx")
	return err == nil
}

func isNginxRunning() bool {
	cmd := exec.Command("systemctl", "is-active", "nginx")
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) == "active"
}

func getNginxVersion() string {
	cmd := exec.Command("nginx", "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`nginx/([\d.]+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1]
	}
	return strings.TrimSpace(string(output))
}

func getNginxActiveConns() int {
	resp, err := http.Get("http://127.0.0.1/nginx_status")
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	re := regexp.MustCompile(`Active connections:\s*(\d+)`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		n, _ := strconv.Atoi(matches[1])
		return n
	}
	return 0
}

func generateSiteNginxConf(site *model.NginxSite) string {
	confDir := "/etc/nginx/conf.d"
	os.MkdirAll(confDir, 0755)
	confPath := filepath.Join(confDir, fmt.Sprintf("%s.conf", site.Domain))

	if site.Status == "stopped" {
		os.Remove(confPath)
		return confPath
	}

	tmplStr := buildNginxConfig(site)
	os.WriteFile(confPath, []byte(tmplStr), 0644)

	return confPath
}

func buildNginxConfig(site *model.NginxSite) string {
	var buf strings.Builder

	if site.SSL {
		buf.WriteString(fmt.Sprintf("server {\n    listen %d;\n    server_name %s;\n    return 301 https://$host$request_uri;\n}\n\n", site.Port, site.Domain))
		buf.WriteString(fmt.Sprintf("server {\n    listen 443 ssl http2;\n    server_name %s;\n", site.Domain))
		buf.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", site.SSLCert))
		buf.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", site.SSLKey))
		buf.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
		buf.WriteString("    ssl_ciphers HIGH:!aNULL:!MD5;\n\n")
	} else {
		buf.WriteString(fmt.Sprintf("server {\n    listen %d;\n    server_name %s;\n\n", site.Port, site.Domain))
	}

	if site.Gzip {
		buf.WriteString("    gzip on;\n    gzip_types text/plain text/css application/json application/javascript text/xml application/xml;\n    gzip_min_length 1024;\n\n")
	}

	switch site.ProxyType {
	case "proxy":
		buf.WriteString(fmt.Sprintf("    root %s;\n    index index.html index.htm;\n\n", site.Root))
		buf.WriteString(fmt.Sprintf("    location / {\n        proxy_pass %s;\n        proxy_set_header Host $host;\n        proxy_set_header X-Real-IP $remote_addr;\n        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n        proxy_set_header X-Forwarded-Proto $scheme;\n        proxy_http_version 1.1;\n        proxy_set_header Upgrade $http_upgrade;\n        proxy_set_header Connection \"upgrade\";\n    }\n", site.ProxyPass))
	default:
		buf.WriteString(fmt.Sprintf("    root %s;\n    index index.html index.htm;\n\n", site.Root))
		buf.WriteString("    location / {\n        try_files $uri $uri/ /index.html;\n    }\n")
	}

	buf.WriteString(fmt.Sprintf("\n    access_log /var/log/nginx/%s_access.log;\n", site.Domain))
	buf.WriteString(fmt.Sprintf("    error_log /var/log/nginx/%s_error.log;\n", site.Domain))
	buf.WriteString("}\n")

	return buf.String()
}

func acmeInstallAndCert(site *model.NginxSite, email string) {
	addLog("info", "nginx", "开始申请 SSL 证书: "+site.Domain)

	acmePath := os.Getenv("HOME") + "/.acme.sh/acme.sh"
	if _, err := os.Stat(acmePath); os.IsNotExist(err) {
		cmd := exec.Command("bash", "-c", fmt.Sprintf("curl https://get.acme.sh | sh -s email=%s", email))
		cmd.Run()
	}

	cmd := exec.Command(acmePath, "--issue", "-d", site.Domain, "--webroot", site.Root)
	output, err := cmd.CombinedOutput()
	if err != nil {
		addLog("error", "nginx", "SSL 证书申请失败: "+string(output))
		return
	}

	certDir := fmt.Sprintf("/etc/nginx/ssl/%s", site.Domain)
	os.MkdirAll(certDir, 0755)
	certPath := filepath.Join(certDir, "fullchain.pem")
	keyPath := filepath.Join(certDir, "privkey.pem")

	cmd = exec.Command(acmePath, "--installcert", "-d", site.Domain, "--cert-file", certPath, "--key-file", keyPath, "--reloadcmd", "systemctl reload nginx")
	cmd.Run()

	config.DB.Model(site).Updates(map[string]interface{}{
		"ssl":      true,
		"ssl_cert": certPath,
		"ssl_key":  keyPath,
	})

	generateSiteNginxConf(site)
	reloadNginx()
	addLog("info", "nginx", "SSL 证书申请成功: "+site.Domain)
}

func acmeRenewCert(site *model.NginxSite) {
	acmePath := os.Getenv("HOME") + "/.acme.sh/acme.sh"
	cmd := exec.Command(acmePath, "--renew", "-d", site.Domain, "--force")
	output, err := cmd.CombinedOutput()
	if err != nil {
		addLog("error", "nginx", "SSL 证书续签失败: "+string(output))
		return
	}
	addLog("info", "nginx", "SSL 证书续签成功: "+site.Domain)
}

func testNginxAndReload() {
	cmd := exec.Command("nginx", "-t")
	if err := cmd.Run(); err == nil {
		reloadNginx()
	}
}

// NginxImportSites 从现有 Nginx 配置导入站点
func NginxImportSites(c *gin.Context) {
	confDir := "/etc/nginx/conf.d"
	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		confDir = "/etc/nginx/sites-enabled"
	}

	files, err := filepath.Glob(filepath.Join(confDir, "*.conf"))
	if err != nil {
		fail(c, 500, "读取配置目录失败")
		return
	}

	imported := 0
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		conf := string(content)
		domainRe := regexp.MustCompile(`server_name\s+([^;]+);`)
		rootRe := regexp.MustCompile(`root\s+([^;]+);`)

		domainMatch := domainRe.FindStringSubmatch(conf)
		rootMatch := rootRe.FindStringSubmatch(conf)

		if len(domainMatch) < 2 {
			continue
		}

		domain := strings.TrimSpace(domainMatch[1])
		root := ""
		if len(rootMatch) > 1 {
			root = strings.TrimSpace(rootMatch[1])
		}

		var count int64
		config.DB.Model(&model.NginxSite{}).Where("domain = ?", domain).Count(&count)
		if count > 0 {
			continue
		}

		name := strings.Split(domain, ".")[0]
		site := model.NginxSite{
			Name:       name,
			Domain:     domain,
			Root:       root,
			Port:       80,
			ProxyType:  "static",
			Status:     "running",
			ConfigFile: file,
		}

		if strings.Contains(conf, "ssl_certificate") {
			site.SSL = true
			certRe := regexp.MustCompile(`ssl_certificate\s+([^;]+);`)
			keyRe := regexp.MustCompile(`ssl_certificate_key\s+([^;]+);`)
			if m := certRe.FindStringSubmatch(conf); len(m) > 1 {
				site.SSLCert = strings.TrimSpace(m[1])
			}
			if m := keyRe.FindStringSubmatch(conf); len(m) > 1 {
				site.SSLKey = strings.TrimSpace(m[1])
			}
		}

		proxyRe := regexp.MustCompile(`proxy_pass\s+([^;]+);`)
		if m := proxyRe.FindStringSubmatch(conf); len(m) > 1 {
			site.ProxyType = "proxy"
			site.ProxyPass = strings.TrimSpace(m[1])
		}

		config.DB.Create(&site)
		imported++
	}

	addLog("info", "nginx", fmt.Sprintf("导入 %d 个站点配置", imported))
	success(c, gin.H{"imported": imported})
}

// NginxSiteReload 重载单个站点
func NginxSiteReload(c *gin.Context) {
	id := c.Param("id")
	var site model.NginxSite
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	confPath := generateSiteNginxConf(&site)
	content, _ := os.ReadFile(confPath)
	config.DB.Model(&site).Updates(map[string]interface{}{
		"config_file":    confPath,
		"config_content": string(content),
	})

	testNginxAndReload()
	success(c, gin.H{"message": "站点已重载"})
}
