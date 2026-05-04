package handler

import (
	"fmt"
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CPU 使用率采集（基于 /proc/stat 差值计算）
var (
	lastCPUTotal  int64
	lastCPUIdle   int64
	cpuMu         sync.Mutex
)

func init() {
	idle, total := readCPUTimes()
	lastCPUIdle = idle
	lastCPUTotal = total
}

// GetDashboard 获取仪表盘概览
// @Summary 获取仪表盘概览数据
// @Description 返回网站数、数据库数、容器数等统计信息
// @Tags 仪表盘
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard [get]
func GetDashboard(c *gin.Context) {
	var siteCount, dbCount, containerCount, taskCount, sshCount, ruleCount int64
	config.DB.Model(&model.Website{}).Count(&siteCount)
	config.DB.Model(&model.Database{}).Count(&dbCount)
	config.DB.Model(&model.Container{}).Count(&containerCount)
	config.DB.Model(&model.Task{}).Count(&taskCount)
	config.DB.Model(&model.SSHAccount{}).Count(&sshCount)
	config.DB.Model(&model.SecurityRule{}).Count(&ruleCount)

	// 统计运行中的
	var siteRunning, dbRunning, containerRunning int64
	config.DB.Model(&model.Website{}).Where("status = ?", "running").Count(&siteRunning)
	config.DB.Model(&model.Database{}).Where("status = ?", "running").Count(&dbRunning)
	config.DB.Model(&model.Container{}).Where("status = ?", "running").Count(&containerRunning)

	success(c, gin.H{
		"websites":          siteCount,
		"websites_running":  siteRunning,
		"databases":         dbCount,
		"databases_running": dbRunning,
		"containers":        containerCount,
		"containers_running": containerRunning,
		"tasks":             taskCount,
		"ssh_accounts":      sshCount,
		"security_rules":    ruleCount,
		"panel_version":     config.AppConfig.Panel.Version,
		"server_time":       time.Now().Format("2006-01-02 15:04:05"),
	})
}

// GetSystemInfo 获取系统信息
// @Summary 获取服务器系统信息
// @Description 返回操作系统、架构、CPU 核心数、主机名、运行时间、内核版本
// @Tags 仪表盘
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard/system-info [get]
func GetSystemInfo(c *gin.Context) {
	info := gin.H{
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"go_version": runtime.Version(),
		"num_cpu":   runtime.NumCPU(),
		"hostname":  getHostname(),
		"uptime":    getUptime(),
		"kernel":    getKernelVersion(),
	}
	success(c, info)
}

// GetSystemStatus 获取系统实时状态
// @Summary 获取 CPU、内存、磁盘实时状态
// @Tags 仪表盘
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard/system-status [get]
func GetSystemStatus(c *gin.Context) {
	loadAvg := getLoadAvg()
	memInfo := getMemInfo()
	diskInfo := getDiskInfo()
	cpuPercent := getCPUPercent()

	success(c, gin.H{
		"load": loadAvg,
		"cpu": gin.H{
			"cores":        runtime.NumCPU(),
			"used_percent": cpuPercent,
		},
		"memory": memInfo,
		"disk":   diskInfo,
	})
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

func getUptime() string {
	if runtime.GOOS == "windows" {
		return "N/A"
	}
	out, err := exec.Command("uptime", "-p").Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(out))
}

func getKernelVersion() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("Windows %s", runtime.GOARCH)
	}
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func getLoadAvg() map[string]float64 {
	if runtime.GOOS == "windows" {
		return map[string]float64{"1m": 0, "5m": 0, "15m": 0}
	}
	out, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return map[string]float64{"1m": 0, "5m": 0, "15m": 0}
	}
	parts := strings.Fields(string(out))
	if len(parts) < 3 {
		return map[string]float64{"1m": 0, "5m": 0, "15m": 0}
	}
	var load1, load5, load15 float64
	fmt.Sscanf(parts[0], "%f", &load1)
	fmt.Sscanf(parts[1], "%f", &load5)
	fmt.Sscanf(parts[2], "%f", &load15)
	return map[string]float64{"1m": load1, "5m": load5, "15m": load15}
}

func getMemInfo() map[string]any {
	if runtime.GOOS == "windows" {
		return map[string]any{
			"total":     0,
			"used":      0,
			"free":      0,
			"used_percent": 0,
		}
	}
	out, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return map[string]any{"total": 0, "used": 0, "free": 0, "used_percent": 0}
	}
	var total, available int64
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d", &total)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d", &available)
		}
	}
	totalMB := total / 1024
	freeMB := available / 1024
	usedMB := totalMB - freeMB
	var usedPercent float64
	if totalMB > 0 {
		usedPercent = float64(usedMB) / float64(totalMB) * 100
	}
	return map[string]any{
		"total":         totalMB,
		"used":          usedMB,
		"free":          freeMB,
		"used_percent":  usedPercent,
	}
}

func getDiskInfo() map[string]any {
	if runtime.GOOS == "windows" {
		return map[string]any{
			"total":        0,
			"used":         0,
			"free":         0,
			"used_percent": 0,
		}
	}
	out, err := exec.Command("df", "-B1", "/").Output()
	if err != nil {
		return map[string]any{"total": 0, "used": 0, "free": 0, "used_percent": 0}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return map[string]any{"total": 0, "used": 0, "free": 0, "used_percent": 0}
	}
	parts := strings.Fields(lines[1])
	if len(parts) < 5 {
		return map[string]any{"total": 0, "used": 0, "free": 0, "used_percent": 0}
	}
	var total, used, free int64
	var usedPercent float64
	fmt.Sscanf(parts[1], "%d", &total)
	fmt.Sscanf(parts[2], "%d", &used)
	fmt.Sscanf(parts[3], "%d", &free)
	fmt.Sscanf(strings.TrimSuffix(parts[4], "%"), "%f", &usedPercent)
	return map[string]any{
		"total":        total / (1024 * 1024),
		"used":         used / (1024 * 1024),
		"free":         free / (1024 * 1024),
		"used_percent": usedPercent,
	}
}

// readCPUTimes 从 /proc/stat 读取 CPU 时间
func readCPUTimes() (idle, total int64) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return 0, 0
	}
	// cpu  user nice system idle iowait irq softirq steal guest guest_nice
	fields := strings.Fields(lines[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0
	}
	var vals []int64
	for i := 1; i < len(fields); i++ {
		var v int64
		fmt.Sscanf(fields[i], "%d", &v)
		vals = append(vals, v)
		total += v
	}
	if len(vals) > 3 {
		idle = vals[3] // idle is 4th field
		if len(vals) > 4 {
			idle += vals[4] // iowait
		}
	}
	return
}

// getCPUPercent 计算 CPU 使用率（差值法）
func getCPUPercent() float64 {
	cpuMu.Lock()
	defer cpuMu.Unlock()

	idle, total := readCPUTimes()
	if total == 0 {
		return 0
	}

	totalDelta := total - lastCPUTotal
	idleDelta := idle - lastCPUIdle

	lastCPUTotal = total
	lastCPUIdle = idle

	if lastCPUTotal == 0 || totalDelta == 0 {
		return 0
	}
	return (1.0 - float64(idleDelta)/float64(totalDelta)) * 100
}
