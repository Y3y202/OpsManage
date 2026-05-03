package handler

import (
	"fmt"
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/gin-gonic/gin"
)

const nginxConfTpl = `server {
    listen {{.Port}};
    server_name {{.Domain}};
    root {{.Path}};
    index index.html index.htm index.php;

    {{if .WAFEnabled}}
    # WAF - ModSecurity
    modsecurity on;
    modsecurity_rules_file /etc/nginx/modsecurity/modsecurity.conf;
    {{end}}

    {{if .SSLEnabled}}
    listen 443 ssl http2;
    ssl_certificate {{.SSLCertPath}};
    ssl_certificate_key {{.SSLKeyPath}};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    if ($scheme != "https") {
        return 301 https://$host$request_uri;
    }
    {{end}}

    location / {
        try_files $uri $uri/ /index.html;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/var/run/php/php-fpm.sock;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
    }

    location ~ /\.ht {
        deny all;
    }

    access_log /var/log/nginx/{{.Domain}}_access.log;
    error_log /var/log/nginx/{{.Domain}}_error.log;
}
`

const nginxConfDir = "/etc/nginx/conf.d"

// ListWebsites 获取网站列表
// @Summary 获取网站列表（分页）
// @Tags 网站管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /websites [get]
func ListWebsites(c *gin.Context) {
	var sites []model.Website
	var total int64
	query := config.DB.Model(&model.Website{})
	query.Count(&total)
	paginate(c, query).Order("id desc").Find(&sites)
	pageResult(c, sites, total)
}

type CreateWebsiteReq struct {
	Name       string `json:"name" binding:"required"`
	Domain     string `json:"domain" binding:"required"`
	Path       string `json:"path" binding:"required"`
	Port       int    `json:"port"`
	Remark     string `json:"remark"`
	WAFEnabled bool   `json:"waf_enabled"`
	WAFRules   string `json:"waf_rules"`
}

// CreateWebsite 创建网站
// @Summary 创建网站
// @Description 创建网站并自动生成 Nginx 配置文件
// @Tags 网站管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateWebsiteReq true "网站信息"
// @Success 200 {object} map[string]interface{}
// @Router /websites [post]
func CreateWebsite(c *gin.Context) {
	var req CreateWebsiteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	if req.Port == 0 {
		req.Port = 80
	}

	os.MkdirAll(req.Path, 0755)

	site := model.Website{
		Name:        req.Name,
		Domain:      req.Domain,
		Path:        req.Path,
		Port:        req.Port,
		Status:      "running",
		WAFEnabled:  req.WAFEnabled,
		WAFRules:    req.WAFRules,
		Remark:      req.Remark,
	}

	if err := config.DB.Create(&site).Error; err != nil {
		fail(c, 500, "创建失败")
		return
	}

	addLog("info", "website", "创建网站: "+site.Name+" ("+site.Domain+")")

	confPath := generateNginxConf(&site)
	site.NginxConf = confPath
	config.DB.Model(&site).Update("nginx_conf", confPath)

	reloadNginx()

	success(c, site)
}

// GetWebsite 获取网站详情
// @Summary 获取网站详情
// @Tags 网站管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "网站ID"
// @Success 200 {object} map[string]interface{}
// @Router /websites/{id} [get]
func GetWebsite(c *gin.Context) {
	id := c.Param("id")
	var site model.Website
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, site)
}

type UpdateWebsiteReq struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	Path       string `json:"path"`
	Port       int    `json:"port"`
	Remark     string `json:"remark"`
	WAFEnabled *bool  `json:"waf_enabled"`
	WAFRules   string `json:"waf_rules"`
	SSLEnabled *bool  `json:"ssl_enabled"`
	SSLCertPath string `json:"ssl_cert_path"`
	SSLKeyPath  string `json:"ssl_key_path"`
}

// UpdateWebsite 更新网站
// @Summary 更新网站配置
// @Description 更新网站信息并重新生成 Nginx 配置
// @Tags 网站管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "网站ID"
// @Param body body UpdateWebsiteReq true "更新信息"
// @Success 200 {object} map[string]interface{}
// @Router /websites/{id} [put]
func UpdateWebsite(c *gin.Context) {
	id := c.Param("id")
	var site model.Website
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var req UpdateWebsiteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Domain != "" {
		updates["domain"] = req.Domain
	}
	if req.Path != "" {
		updates["path"] = req.Path
	}
	if req.Port > 0 {
		updates["port"] = req.Port
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}
	if req.WAFEnabled != nil {
		updates["waf_enabled"] = *req.WAFEnabled
	}
	if req.WAFRules != "" {
		updates["waf_rules"] = req.WAFRules
	}
	if req.SSLEnabled != nil {
		updates["ssl_enabled"] = *req.SSLEnabled
	}
	if req.SSLCertPath != "" {
		updates["ssl_cert_path"] = req.SSLCertPath
	}
	if req.SSLKeyPath != "" {
		updates["ssl_key_path"] = req.SSLKeyPath
	}

	config.DB.Model(&site).Updates(updates)
	config.DB.First(&site, id)

	confPath := generateNginxConf(&site)
	site.NginxConf = confPath
	config.DB.Model(&site).Update("nginx_conf", confPath)
	reloadNginx()

	success(c, site)
}

// DeleteWebsite 删除网站
// @Summary 删除网站
// @Description 删除网站并移除 Nginx 配置
// @Tags 网站管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "网站ID"
// @Success 200 {object} map[string]interface{}
// @Router /websites/{id} [delete]
func DeleteWebsite(c *gin.Context) {
	id := c.Param("id")
	var site model.Website
	if err := config.DB.First(&site, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	removeNginxConf(&site)
	reloadNginx()
	config.DB.Delete(&site)
	addLog("info", "website", "删除网站: "+site.Name+" ("+site.Domain+")")
	success(c, nil)
}

// StartWebsite 启动网站
// @Summary 启动网站
// @Tags 网站管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "网站ID"
// @Success 200 {object} map[string]interface{}
// @Router /websites/{id}/start [post]
func StartWebsite(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Website{}).Where("id = ?", id).Update("status", "running")
	reloadNginx()
	success(c, gin.H{"status": "running"})
}

// StopWebsite 停止网站
// @Summary 停止网站
// @Tags 网站管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "网站ID"
// @Success 200 {object} map[string]interface{}
// @Router /websites/{id}/stop [post]
func StopWebsite(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Website{}).Where("id = ?", id).Update("status", "stopped")
	reloadNginx()
	success(c, gin.H{"status": "stopped"})
}

func generateNginxConf(site *model.Website) string {
	confName := fmt.Sprintf("%s.conf", site.Domain)
	confPath := filepath.Join(nginxConfDir, confName)

	tmpl, err := template.New("nginx").Parse(nginxConfTpl)
	if err != nil {
		return ""
	}

	f, err := os.Create(confPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	tmpl.Execute(f, site)
	return confPath
}

func removeNginxConf(site *model.Website) {
	confName := fmt.Sprintf("%s.conf", site.Domain)
	confPath := filepath.Join(nginxConfDir, confName)
	os.Remove(confPath)
}

func reloadNginx() {
	exec.Command("nginx", "-t").Run()
	exec.Command("nginx", "-s", "reload").Run()
}
