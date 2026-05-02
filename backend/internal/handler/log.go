package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"

	"github.com/gin-gonic/gin"
)

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

func GetLogSources(c *gin.Context) {
	var sources []string
	config.DB.Model(&model.LogEntry{}).Distinct().Pluck("source", &sources)
	success(c, sources)
}

func ClearLogs(c *gin.Context) {
	source := c.Query("source")
	if source != "" {
		config.DB.Where("source = ?", source).Delete(&model.LogEntry{})
	} else {
		config.DB.Where("1 = 1").Delete(&model.LogEntry{})
	}
	success(c, nil)
}

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
