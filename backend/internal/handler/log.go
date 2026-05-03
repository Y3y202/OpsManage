package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"

	"github.com/gin-gonic/gin"
)

// ListLogs 获取操作日志
// @Summary 获取操作日志列表（分页、可筛选）
// @Tags 日志管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param level query string false "日志级别 (info/warn/error)"
// @Param source query string false "日志来源"
// @Param keyword query string false "关键词搜索"
// @Success 200 {object} map[string]interface{}
// @Router /logs [get]
func ListLogs(c *gin.Context) {
	var logs []model.LogEntry
	var total int64
	query := config.DB.Model(&model.LogEntry{})

	if level := c.Query("level"); level != "" {
		query = query.Where("level = ?", level)
	}
	if source := c.Query("source"); source != "" {
		query = query.Where("source = ?", source)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("message LIKE ?", "%"+keyword+"%")
	}

	query.Count(&total)
	paginate(c, query).Order("id desc").Find(&logs)
	pageResult(c, logs, total)
}

// GetLogSources 获取日志来源列表
// @Summary 获取所有日志来源
// @Tags 日志管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /logs/sources [get]
func GetLogSources(c *gin.Context) {
	var sources []string
	config.DB.Model(&model.LogEntry{}).Distinct().Pluck("source", &sources)
	success(c, sources)
}

// ClearLogs 清空日志
// @Summary 清空操作日志
// @Description 按来源清空或清空全部
// @Tags 日志管理
// @Produce json
// @Security BearerAuth
// @Param source query string false "日志来源（为空则清空全部）"
// @Success 200 {object} map[string]interface{}
// @Router /logs/clear [delete]
func ClearLogs(c *gin.Context) {
	source := c.Query("source")
	if source != "" {
		config.DB.Where("source = ?", source).Delete(&model.LogEntry{})
	} else {
		config.DB.Where("1 = 1").Delete(&model.LogEntry{})
	}
	success(c, nil)
}

// GetSystemLogs 获取系统日志
// @Summary 获取系统日志文件内容
// @Description 支持 syslog, auth, nginx, nginx_error
// @Tags 日志管理
// @Produce json
// @Security BearerAuth
// @Param type query string false "日志类型" Enums(syslog, auth, nginx, nginx_error) default(syslog)
// @Param lines query int false "行数" default(200)
// @Success 200 {object} map[string]interface{}
// @Router /logs/system [get]
func GetSystemLogs(c *gin.Context) {
	logType := c.DefaultQuery("type", "syslog")
	lines := c.DefaultQuery("lines", "200")

	var logFile string
	switch logType {
	case "syslog":
		logFile = "/var/log/syslog"
	case "auth":
		logFile = "/var/log/auth.log"
	case "nginx":
		logFile = "/var/log/nginx/access.log"
	case "nginx_error":
		logFile = "/var/log/nginx/error.log"
	default:
		logFile = "/var/log/syslog"
	}

	out, err := readLastLines(logFile, lines)
	if err != nil {
		fail(c, 500, "读取日志失败: "+err.Error())
		return
	}
	success(c, gin.H{
		"type":    logType,
		"file":    logFile,
		"content": out,
	})
}
