package handler

import (
	"fmt"
	"os/exec"
	"runtime"
)

func readLastLines(filePath string, lines string) (string, error) {
	if runtime.GOOS == "windows" {
		return "日志查看仅支持Linux系统", nil
	}
	out, err := exec.Command("tail", "-n", lines, filePath).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("读取日志失败: %s", string(out))
	}
	return string(out), nil
}
