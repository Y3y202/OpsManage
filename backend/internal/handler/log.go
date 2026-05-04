package handler

import (
	"fmt"
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os/exec"
	"regexp"
	"strings"

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

// SSHLogEntry SSH 登录日志条目
type SSHLogEntry struct {
	Time    string `json:"time"`
	Event   string `json:"event"`
	User    string `json:"user"`
	IP      string `json:"ip"`
	Port    string `json:"port"`
	Message string `json:"message"`
}

var (
	reAccepted = regexp.MustCompile(`Accepted \S+ for (\S+) from (\S+) port (\d+)`)
	reFailed   = regexp.MustCompile(`Failed \S+ for (\S+) from (\S+) port (\d+)`)
	reClosed   = regexp.MustCompile(`Connection closed by authenticating user (\S+) (\S+) port (\d+)`)
	reClosedIP = regexp.MustCompile(`Connection closed by (\S+) port (\d+)`)
	reSessionO = regexp.MustCompile(`session opened for user (\S+)`)
	reSessionC = regexp.MustCompile(`session closed for user (\S+)`)
	reAuthFail = regexp.MustCompile(`authentication failure.*rhost=(\S+).*user=(\S+)`)
)

// GetSSHLogs 获取 SSH 登录日志（从 auth.log 解析）
// @Summary 获取 SSH 登录日志
// @Description 从系统 auth.log 解析 SSH 登录记录，支持筛选
// @Tags 日志管理
// @Produce json
// @Security BearerAuth
// @Param lines query int false "读取行数" default(500)
// @Param event query string false "事件类型筛选 (accepted/failed/closed)"
// @Param keyword query string false "关键词搜索"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /logs/ssh [get]
func GetSSHLogs(c *gin.Context) {
	lines := c.DefaultQuery("lines", "500")
	filterEvent := c.Query("event")
	keyword := c.Query("keyword")

	// 尝试多个常见日志路径
	logFile := "/var/log/auth.log"
	var out []byte
	var err error
	out, err = exec.Command("tail", "-n", lines, logFile).CombinedOutput()
	if err != nil {
		logFile = "/var/log/secure"
		out, err = exec.Command("tail", "-n", lines, logFile).CombinedOutput()
		if err != nil {
			fail(c, 500, fmt.Sprintf("读取 SSH 日志失败: %s", string(out)))
			return
		}
	}

	rawLines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var entries []SSHLogEntry

	for _, line := range rawLines {
		if !strings.Contains(line, "sshd") {
			continue
		}

		entry := parseSSHLogLine(line)
		if entry == nil {
			continue
		}

		// 筛选事件类型
		if filterEvent != "" && entry.Event != filterEvent {
			continue
		}
		// 关键词搜索
		if keyword != "" && !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(keyword)) &&
			!strings.Contains(strings.ToLower(entry.User), strings.ToLower(keyword)) &&
			!strings.Contains(strings.ToLower(entry.IP), strings.ToLower(keyword)) {
			continue
		}

		entries = append(entries, *entry)
	}

	// 反转顺序（最新的在前）
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// 分页
	total := len(entries)
	pageNum := 1
	fmt.Sscanf(c.DefaultQuery("page", "1"), "%d", &pageNum)
	pageSize := 20
	fmt.Sscanf(c.DefaultQuery("page_size", "20"), "%d", &pageSize)
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (pageNum - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	pageResult(c, entries[start:end], int64(total))
}

func parseSSHLogLine(line string) *SSHLogEntry {
	// 提取时间戳（取第一个字段）
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil
	}
	timeStr := parts[0]

	// 提取消息部分（sshd[PID]: 之后的内容）
	msgIdx := strings.Index(line, "sshd[")
	if msgIdx == -1 {
		return nil
	}
	msgPart := line[msgIdx:]
	if colonIdx := strings.Index(msgPart, ": "); colonIdx != -1 {
		msgPart = msgPart[colonIdx+2:]
	}

	entry := &SSHLogEntry{Time: timeStr, Message: msgPart}

	if m := reAccepted.FindStringSubmatch(msgPart); m != nil {
		entry.Event = "accepted"
		entry.User = m[1]
		entry.IP = m[2]
		entry.Port = m[3]
		return entry
	}
	if m := reFailed.FindStringSubmatch(msgPart); m != nil {
		entry.Event = "failed"
		entry.User = m[1]
		entry.IP = m[2]
		entry.Port = m[3]
		return entry
	}
	if m := reClosed.FindStringSubmatch(msgPart); m != nil {
		entry.Event = "closed"
		entry.User = m[1]
		entry.IP = m[2]
		entry.Port = m[3]
		return entry
	}
	if m := reClosedIP.FindStringSubmatch(msgPart); m != nil {
		entry.Event = "closed"
		entry.IP = m[1]
		entry.Port = m[2]
		return entry
	}
	if m := reAuthFail.FindStringSubmatch(msgPart); m != nil {
		entry.Event = "failed"
		entry.User = m[2]
		entry.IP = m[1]
		return entry
	}
	if m := reSessionO.FindStringSubmatch(msgPart); m != nil {
		entry.Event = "session_open"
		entry.User = m[1]
		return entry
	}
	if m := reSessionC.FindStringSubmatch(msgPart); m != nil {
		entry.Event = "session_close"
		entry.User = m[1]
		return entry
	}

	// 其他 sshd 消息也返回，归类为 other
	entry.Event = "other"
	return entry
}
