package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"opsmanage/internal/config"
	"opsmanage/internal/model"

	"github.com/gin-gonic/gin"
)

// ==================== 证书申请 (Let's Encrypt) ====================

type applyCertReq struct {
	Domain    string `json:"domain" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Challenge string `json:"challenge"` // http / dns
	// HTTP 验证参数
	Standalone bool   `json:"standalone"`
	WebRoot    string `json:"web_root"`
	// DNS 验证参数
	DnsProvider string `json:"dns_provider"` // aliyun / tencent / huawei / cloudflare
	DnsKey      string `json:"dns_key"`      // AccessKey / SecretId / AK / CF_Key
	DnsSecret   string `json:"dns_secret"`   // AccessSecret / SecretKey / SK / CF_Email
	// 华为云额外参数
	HwProjectId string `json:"hw_project_id"`
}

func ApplyCertificate(c *gin.Context) {
	var req applyCertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 500, "参数错误: "+err.Error())
		return
	}

	// 默认 HTTP 验证
	if req.Challenge == "" {
		req.Challenge = "http"
	}

	// 泛域名必须使用 DNS 验证
	if strings.HasPrefix(req.Domain, "*.") && req.Challenge != "dns" {
		fail(c, 500, "泛域名证书必须使用 DNS 验证方式")
		return
	}

	// 检查 acme.sh 是否可用
	acmeSh := os.Getenv("HOME") + "/.acme.sh/acme.sh"
	if _, err := os.Stat(acmeSh); err != nil {
		// 尝试标准路径
		acmeSh = "/root/.acme.sh/acme.sh"
		if _, err := os.Stat(acmeSh); err != nil {
			fail(c, 500, "acme.sh 未安装")
			return
		}
	}

	var args []string
	var envVars []string

	if req.Challenge == "dns" {
		// DNS-01 验证模式
		if req.DnsProvider == "" {
			fail(c, 500, "请选择 DNS 提供商")
			return
		}
		if req.DnsKey == "" || req.DnsSecret == "" {
			fail(c, 500, "请填写 DNS 提供商的 AccessKey 和 Secret")
			return
		}

		// 设置 DNS 提供商环境变量
		switch req.DnsProvider {
		case "aliyun":
			envVars = append(envVars, "Ali_Key="+req.DnsKey, "Ali_Secret="+req.DnsSecret)
			args = append(args, "--dns", "dns_ali")
		case "tencent":
			envVars = append(envVars, "Tencent_SecretId="+req.DnsKey, "Tencent_SecretKey="+req.DnsSecret)
			args = append(args, "--dns", "dns_tencent")
		case "huawei":
			if req.HwProjectId == "" {
				fail(c, 500, "华为云需要填写项目 ID")
				return
			}
			envVars = append(envVars,
				"HUAWEICLOUD_AK="+req.DnsKey,
				"HUAWEICLOUD_SK="+req.DnsSecret,
				"HUAWEICLOUD_ProjectID="+req.HwProjectId,
			)
			args = append(args, "--dns", "dns_huaweicloud")
		case "cloudflare":
			envVars = append(envVars, "CF_Key="+req.DnsKey, "CF_Email="+req.DnsSecret)
			args = append(args, "--dns", "dns_cf")
		default:
			fail(c, 500, "不支持的 DNS 提供商: "+req.DnsProvider)
			return
		}

		args = append(args,
			"--issue",
			"--domain", req.Domain,
			"--email", req.Email,
			"--force",
		)
	} else {
		// HTTP 验证模式
		args = append(args,
			"--issue",
			"--domain", req.Domain,
			"--email", req.Email,
		)
		if req.Standalone {
			args = append(args, "--standalone")
		} else {
			webRoot := req.WebRoot
			if webRoot == "" {
				webRoot = "/var/www/html"
			}
			args = append(args, "--webroot", webRoot)
		}
	}

	// 安装证书到固定目录
	certName := req.Domain
	certName = strings.ReplaceAll(certName, "*", "wildcard")
	certDir := fmt.Sprintf("/etc/opsmanage/certs/%s", certName)
	os.MkdirAll(certDir, 0700)

	args = append(args,
		"--cert-file", filepath.Join(certDir, "cert.pem"),
		"--key-file", filepath.Join(certDir, "privkey.pem"),
		"--fullchain-file", filepath.Join(certDir, "fullchain.pem"),
		"--ca-file", filepath.Join(certDir, "chain.pem"),
	)

	// 执行 acme.sh
	cmd := exec.Command(acmeSh, args...)
	cmd.Env = append(os.Environ(), envVars...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		config.DB.Create(&model.LogEntry{
			Level:   "error",
			Source:  "certificate",
			Message: "证书申请失败: " + req.Domain,
			Detail:  outputStr,
		})
		fail(c, 500, "证书申请失败: "+extractAcmeError(outputStr))
		return
	}

	certPath := filepath.Join(certDir, "fullchain.pem")
	keyPath := filepath.Join(certDir, "privkey.pem")

	// 检查证书文件是否存在
	if _, err := os.Stat(certPath); err != nil {
		fail(c, 500, "证书文件未生成，请检查日志")
		return
	}

	// 读取证书信息
	info := parseCertInfo(certPath)
	now := time.Now()
	status := "valid"
	if info.NotAfter.Before(now) {
		status = "expired"
	} else if info.NotAfter.Before(now.Add(30 * 24 * time.Hour)) {
		status = "about_to_expire"
	}

	certType := "letsencrypt"
	if req.Challenge == "dns" {
		certType = "letsencrypt-dns"
	}

	cert := model.Certificate{
		Name:      certName,
		Domain:    req.Domain,
		Type:      certType,
		CertPath:  certPath,
		KeyPath:   keyPath,
		ChainPath: filepath.Join(certDir, "chain.pem"),
		Issuer:    info.Issuer,
		NotBefore: info.NotBefore,
		NotAfter:  info.NotAfter,
		Subject:   info.Subject,
		SANs:      info.SANs,
		Status:    status,
	}

	// 如果已存在同名证书则更新
	var existing model.Certificate
	if err := config.DB.Where("name = ?", certName).First(&existing).Error; err == nil {
		cert.ID = existing.ID
		config.DB.Save(&cert)
	} else {
		config.DB.Create(&cert)
	}

	config.DB.Create(&model.LogEntry{
		Level:   "info",
		Source:  "certificate",
		Message: "证书申请成功: " + req.Domain,
		Detail:  outputStr,
	})

	success(c, cert)
}

// ==================== 上传自定义证书 ====================

type uploadCertReq struct {
	Name     string `json:"name" binding:"required"`
	Domain   string `json:"domain" binding:"required"`
	Cert     string `json:"cert" binding:"required"`     // PEM 格式证书内容
	Key      string `json:"key" binding:"required"`       // PEM 格式私钥内容
	Chain    string `json:"chain"`                         // 可选：证书链
}

func UploadCertificate(c *gin.Context) {
	var req uploadCertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 500, "参数错误: "+err.Error())
		return
	}

	// 检查重名
	var existing model.Certificate
	if err := config.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		fail(c, 500, "证书名称已存在")
		return
	}

	// 创建证书目录
	certDir := fmt.Sprintf("/etc/opsmanage/certs/%s", req.Name)
	if err := os.MkdirAll(certDir, 0700); err != nil {
		fail(c, 500, "创建证书目录失败: "+err.Error())
		return
	}

	// 写入证书文件
	certPath := filepath.Join(certDir, "fullchain.pem")
	keyPath := filepath.Join(certDir, "privkey.pem")
	chainPath := filepath.Join(certDir, "chain.pem")

	if err := os.WriteFile(certPath, []byte(req.Cert), 0600); err != nil {
		fail(c, 500, "写入证书文件失败: "+err.Error())
		return
	}
	if err := os.WriteFile(keyPath, []byte(req.Key), 0600); err != nil {
		fail(c, 500, "写入私钥文件失败: "+err.Error())
		return
	}
	if req.Chain != "" {
		os.WriteFile(chainPath, []byte(req.Chain), 0600)
	}

	// 解析证书信息
	info := parseCertInfo(certPath)
	now := time.Now()
	status := "valid"
	if info.NotAfter.Before(now) {
		status = "expired"
	} else if info.NotAfter.Before(now.Add(30 * 24 * time.Hour)) {
		status = "about_to_expire"
	}

	cert := model.Certificate{
		Name:      req.Name,
		Domain:    req.Domain,
		Type:      "custom",
		CertPath:  certPath,
		KeyPath:   keyPath,
		ChainPath: chainPath,
		Issuer:    info.Issuer,
		NotBefore: info.NotBefore,
		NotAfter:  info.NotAfter,
		Subject:   info.Subject,
		SANs:      info.SANs,
		Status:    status,
	}
	config.DB.Create(&cert)

	success(c, cert)
}

// ==================== 证书列表 ====================

func ListCertificates(c *gin.Context) {
	var certs []model.Certificate
	config.DB.Order("created_at desc").Find(&certs)

	// 更新证书状态
	now := time.Now()
	for i := range certs {
		if certs[i].NotAfter.Before(now) {
			certs[i].Status = "expired"
		} else if certs[i].NotAfter.Before(now.Add(30 * 24 * time.Hour)) {
			certs[i].Status = "about_to_expire"
		} else {
			certs[i].Status = "valid"
		}
		config.DB.Model(&certs[i]).Update("status", certs[i].Status)
	}

	success(c, certs)
}

// ==================== 证书详情 ====================

func GetCertificate(c *gin.Context) {
	id := c.Param("id")
	var cert model.Certificate
	if err := config.DB.First(&cert, id).Error; err != nil {
		fail(c, 500, "证书不存在")
		return
	}
	success(c, cert)
}

// ==================== 删除证书 ====================

func DeleteCertificate(c *gin.Context) {
	id := c.Param("id")
	var cert model.Certificate
	if err := config.DB.First(&cert, id).Error; err != nil {
		fail(c, 500, "证书不存在")
		return
	}

	// acme.sh 证书用 revoke 命令撤销
	if cert.Type == "letsencrypt" || cert.Type == "letsencrypt-dns" {
		acmeSh := "/root/.acme.sh/acme.sh"
		exec.Command(acmeSh, "--revoke", "-d", cert.Domain, "--force").Run()
	}

	// 删除证书文件
	certDir := filepath.Dir(cert.CertPath)
	os.RemoveAll(certDir)

	config.DB.Delete(&cert)
	success(c, "证书已删除")
}

// ==================== 续签证书 ====================

func RenewCertificate(c *gin.Context) {
	id := c.Param("id")
	var cert model.Certificate
	if err := config.DB.First(&cert, id).Error; err != nil {
		fail(c, 500, "证书不存在")
		return
	}

	if cert.Type != "letsencrypt" && cert.Type != "letsencrypt-dns" {
		fail(c, 500, "只有 Let's Encrypt 证书支持自动续签")
		return
	}

	acmeSh := "/root/.acme.sh/acme.sh"
	cmd := exec.Command(acmeSh, "--renew", "-d", cert.Domain, "--force")
	output, err := cmd.CombinedOutput()

	if err != nil {
		fail(c, 500, "续签失败: "+string(output))
		return
	}

	// 重新读取证书信息
	info := parseCertInfo(cert.CertPath)
	now := time.Now()
	status := "valid"
	if info.NotAfter.Before(now) {
		status = "expired"
	} else if info.NotAfter.Before(now.Add(30 * 24 * time.Hour)) {
		status = "about_to_expire"
	}

	config.DB.Model(&cert).Updates(map[string]interface{}{
		"issuer":    info.Issuer,
		"not_before": info.NotBefore,
		"not_after":  info.NotAfter,
		"subject":   info.Subject,
		"sans":      info.SANs,
		"status":    status,
	})

	config.DB.Create(&model.LogEntry{
		Level:   "info",
		Source:  "certificate",
		Message: "证书续签成功: " + cert.Domain,
	})

	success(c, "续签成功")
}

// ==================== 查看证书内容 ====================

func GetCertificateContent(c *gin.Context) {
	id := c.Param("id")
	field := c.Param("field") // cert / key / chain

	var cert model.Certificate
	if err := config.DB.First(&cert, id).Error; err != nil {
		fail(c, 500, "证书不存在")
		return
	}

	var path string
	switch field {
	case "cert":
		path = cert.CertPath
	case "key":
		path = cert.KeyPath
	case "chain":
		path = cert.ChainPath
	default:
		fail(c, 500, "无效的字段")
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		fail(c, 500, "读取文件失败: "+err.Error())
		return
	}

	c.JSON(200, gin.H{"code": 200, "data": string(content)})
}

// ==================== 应用证书到站点 ====================

type applyToSiteReq struct {
	WebsiteID uint `json:"website_id" binding:"required"`
}

func ApplyCertToSite(c *gin.Context) {
	id := c.Param("id")
	var cert model.Certificate
	if err := config.DB.First(&cert, id).Error; err != nil {
		fail(c, 500, "证书不存在")
		return
	}

	var req applyToSiteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 500, "参数错误")
		return
	}

	var site model.NginxSite
	if err := config.DB.First(&site, req.WebsiteID).Error; err != nil {
		fail(c, 500, "站点不存在")
		return
	}

	// 更新站点 SSL 配置
	config.DB.Model(&site).Updates(map[string]interface{}{
		"ssl":      true,
		"ssl_cert": cert.CertPath,
		"ssl_key":  cert.KeyPath,
	})

	// 重新生成 nginx 配置
	generateSiteNginxConf(&site)

	success(c, "证书已应用到站点 "+site.Name)
}

// ==================== 工具函数 ====================

type certInfo struct {
	Issuer    string
	NotBefore time.Time
	NotAfter  time.Time
	Subject   string
	SANs      string
}

func parseCertInfo(certPath string) certInfo {
	info := certInfo{}

	// 使用 openssl 读取证书信息
	cmd := exec.Command("openssl", "x509", "-in", certPath, "-noout",
		"-issuer", "-dates", "-subject", "-ext", "subjectAltName")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return info
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "issuer="):
			info.Issuer = strings.TrimPrefix(line, "issuer=")
		case strings.HasPrefix(line, "notBefore="):
			t, _ := time.Parse("Jan 2 15:04:05 2006 MST", strings.TrimPrefix(line, "notBefore="))
			if t.IsZero() {
				t, _ = time.Parse("Jan  2 15:04:05 2006 MST", strings.TrimPrefix(line, "notBefore="))
			}
			info.NotBefore = t
		case strings.HasPrefix(line, "notAfter="):
			t, _ := time.Parse("Jan 2 15:04:05 2006 MST", strings.TrimPrefix(line, "notAfter="))
			if t.IsZero() {
				t, _ = time.Parse("Jan  2 15:04:05 2006 MST", strings.TrimPrefix(line, "notAfter="))
			}
			info.NotAfter = t
		case strings.HasPrefix(line, "subject="):
			info.Subject = strings.TrimPrefix(line, "subject=")
		case strings.HasPrefix(line, "DNS:"):
			sans := []string{}
			for _, part := range strings.Split(line, ",") {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "DNS:") {
					sans = append(sans, strings.TrimPrefix(part, "DNS:"))
				}
			}
			info.SANs = strings.Join(sans, ",")
		}
	}

	return info
}

func extractAcmeError(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error:") ||
			strings.Contains(lower, "failed") ||
			strings.Contains(lower, "problem") {
			return strings.TrimSpace(line)
		}
	}
	// 返回最后几行
	if len(lines) > 3 {
		return strings.TrimSpace(strings.Join(lines[len(lines)-3:], "\n"))
	}
	return strings.TrimSpace(output)
}

// ==================== 前端需要的辅助 API ====================

// ListSitesForCert 返回可绑定的站点列表
func ListSitesForCert(c *gin.Context) {
	var sites []model.NginxSite
	config.DB.Select("id, name, domain, ssl, ssl_cert").Find(&sites)
	success(c, sites)
}

// internal helper - used by handler/common.go, defined here for certificate
func successCert(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"code": 200, "data": data})
}

// 格式化 JSON 用于日志
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
