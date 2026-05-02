package handler

import (
	"fmt"
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetDashboard(c *gin.Context) {
	var siteCount, dbCount, containerCount int64
	config.DB.Model(&model.Website{}).Count(&siteCount)
	config.DB.Model(&model.Database{}).Count(&dbCount)
	config.DB.Model(&model.Container{}).Count(&containerCount)

	success(c, gin.H{
		"websites":        siteCount,
		"databases":       dbCount,
		"containers":      containerCount,
		"panel_version":   config.AppConfig.Panel.Version,
		"server_time":     time.Now().Format("2006-01-02 15:04:05"),
	})
}

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

func GetSystemStatus(c *gin.Context) {
	loadAvg := getLoadAvg()
	memInfo := getMemInfo()
	diskInfo := getDiskInfo()

	success(c, gin.H{
		"load": loadAvg,
		"cpu": gin.H{
			"cores": runtime.NumCPU(),
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
		return fmt.Sprintf("Windows %s", runtime.GOOS)
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
