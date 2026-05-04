package handler

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

// FirewallRule 防火墙规则
type FirewallRule struct {
	ID       int    `json:"id"`
	Chain    string `json:"chain"`    // INPUT / OUTPUT / FORWARD
	Protocol string `json:"protocol"` // tcp / udp / all
	SrcIP    string `json:"src_ip"`
	DstPort  string `json:"dst_port"`
	Target   string `json:"target"` // ACCEPT / DROP / REJECT
	Comment  string `json:"comment"`
}

// FirewallStatus 防火墙状态
type FirewallStatus struct {
	Enabled       bool            `json:"enabled"`
	Backend       string          `json:"backend"` // iptables / ufw / nftables
	Rules         []FirewallRule  `json:"rules"`
	Chains        map[string]int  `json:"chains"` // chain -> rule count
	Fail2BanBans  int             `json:"fail2ban_bans"` // Fail2Ban 封禁数
}

// GetFirewallStatus 获取防火墙状态
// @Summary 获取防火墙状态和规则
// @Tags 防火墙
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /security/firewall/status [get]
func GetFirewallStatus(c *gin.Context) {
	if runtime.GOOS == "windows" {
		success(c, FirewallStatus{Enabled: false, Backend: "none"})
		return
	}

	status := detectFirewallBackend()
	success(c, status)
}

func detectFirewallBackend() FirewallStatus {
	// 优先检测 ufw
	out, err := exec.Command("ufw", "status").CombinedOutput()
	if err == nil && strings.Contains(string(out), "Status: active") {
		return parseUFWStatus(string(out))
	}

	// 检测 nftables
	out, err = exec.Command("nft", "list", "ruleset").CombinedOutput()
	if err == nil && len(out) > 10 {
		return parseNFTStatus(string(out))
	}

	// 降级 iptables
	out, err = exec.Command("iptables", "-L", "-n", "--line-numbers").CombinedOutput()
	if err == nil {
		return parseIPTablesStatus(string(out))
	}

	return FirewallStatus{Enabled: false, Backend: "none", Rules: []FirewallRule{}}
}

func parseUFWStatus(output string) FirewallStatus {
	status := FirewallStatus{Enabled: true, Backend: "ufw", Rules: []FirewallRule{}, Chains: map[string]int{}}
	lines := strings.Split(output, "\n")
	id := 1
	fail2banCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Status:") || strings.HasPrefix(line, "---") || (strings.Contains(line, "To") && strings.Contains(line, "Action")) {
			continue
		}
		// 跳过 Fail2Ban 自动生成规则，单独计数
		if strings.Contains(line, "by Fail2Ban") {
			fail2banCount++
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		rule := FirewallRule{ID: id, Chain: "INPUT", Protocol: "all"}
		rule.DstPort = fields[0]
		if len(fields) > 2 {
			rule.Target = fields[1]
		}
		if len(fields) > 3 {
			rule.SrcIP = fields[3]
		}
		if len(fields) > 5 && strings.HasPrefix(fields[5], "#") {
			rule.Comment = strings.Join(fields[5:], " ")[1:] // 去掉 #
		}
		status.Rules = append(status.Rules, rule)
		status.Chains["INPUT"]++
		id++
	}
	status.Fail2BanBans = fail2banCount
	return status
}

func parseNFTStatus(output string) FirewallStatus {
	status := FirewallStatus{Enabled: true, Backend: "nftables", Rules: []FirewallRule{}, Chains: map[string]int{}}
	lines := strings.Split(output, "\n")
	id := 1
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "chain ") && strings.Contains(line, "{") {
			chain := "INPUT"
			if strings.Contains(line, "output") || strings.Contains(line, "OUTPUT") {
				chain = "OUTPUT"
			} else if strings.Contains(line, "forward") || strings.Contains(line, "FORWARD") {
				chain = "FORWARD"
			}
			status.Chains[chain]++
		}
		if strings.Contains(line, "accept") || strings.Contains(line, "drop") || strings.Contains(line, "reject") {
			rule := FirewallRule{ID: id, Chain: "INPUT", Protocol: "all"}
			if strings.Contains(line, "accept") {
				rule.Target = "ACCEPT"
			} else if strings.Contains(line, "drop") {
				rule.Target = "DROP"
			} else {
				rule.Target = "REJECT"
			}
			if strings.Contains(line, "tcp") {
				rule.Protocol = "tcp"
			} else if strings.Contains(line, "udp") {
				rule.Protocol = "udp"
			}
			if idx := strings.Index(line, "dport "); idx != -1 {
				port := strings.Fields(line[idx+6:])
				if len(port) > 0 {
					rule.DstPort = port[0]
				}
			}
			if idx := strings.Index(line, "saddr "); idx != -1 {
				ip := strings.Fields(line[idx+6:])
				if len(ip) > 0 {
					rule.SrcIP = ip[0]
				}
			}
			status.Rules = append(status.Rules, rule)
			id++
		}
	}
	return status
}

func parseIPTablesStatus(output string) FirewallStatus {
	status := FirewallStatus{Enabled: true, Backend: "iptables", Rules: []FirewallRule{}, Chains: map[string]int{}}
	lines := strings.Split(output, "\n")
	currentChain := ""
	id := 1
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Chain header
		if strings.HasPrefix(line, "Chain ") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				currentChain = parts[1]
			}
			continue
		}
		// Header line
		if strings.Contains(line, "target") && strings.Contains(line, "prot") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		rule := FirewallRule{ID: id, Chain: currentChain}
		rule.Target = fields[0]
		rule.Protocol = fields[1]

		// Find source and destination
		for i, f := range fields {
			if f == "0.0.0.0/0" || f == "::/0" {
				continue
			}
			if i > 1 && fields[i-1] == "ACCEPT" || fields[i-1] == "DROP" || fields[i-1] == "REJECT" {
				continue
			}
		}

		// Extract dpt: port
		for _, f := range fields {
			if strings.HasPrefix(f, "dpt:") {
				rule.DstPort = f[4:]
			}
			if strings.HasPrefix(f, "spt:") {
				continue
			}
		}

		// Source IP
		if len(fields) > 3 {
			src := fields[3]
			if src != "0.0.0.0/0" && src != "::/0" && src != "anywhere" && !strings.HasPrefix(src, "dpt:") && !strings.HasPrefix(src, "spt:") {
				rule.SrcIP = src
			}
		}

		status.Chains[currentChain]++
		status.Rules = append(status.Rules, rule)
		id++
	}
	return status
}

// AddFirewallRule 添加防火墙规则
// @Summary 添加防火墙规则
// @Tags 防火墙
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]string true "规则信息"
// @Success 200 {object} map[string]interface{}
// @Router /security/firewall/rules [post]
func AddFirewallRule(c *gin.Context) {
	var req struct {
		Protocol string `json:"protocol"`
		DstPort  string `json:"dst_port"`
		SrcIP    string `json:"src_ip"`
		Target   string `json:"target"`
		Comment  string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误: "+err.Error())
		return
	}

	if runtime.GOOS == "windows" {
		fail(c, 500, "Windows 不支持防火墙管理")
		return
	}

	// 检测 ufw
	_, err := exec.LookPath("ufw")
	if err == nil {
		args := buildUFWAddArgs(req)
		out, err := exec.Command("ufw", args...).CombinedOutput()
		if err != nil {
			fail(c, 500, fmt.Sprintf("添加规则失败: %s", string(out)))
			return
		}
		success(c, gin.H{"msg": "规则已添加"})
		return
	}

	// 降级 iptables
	args := buildIPTablesAddArgs(req)
	out, err := exec.Command("iptables", args...).CombinedOutput()
	if err != nil {
		fail(c, 500, fmt.Sprintf("添加规则失败: %s", string(out)))
		return
	}
	success(c, gin.H{"msg": "规则已添加"})
}

func buildUFWAddArgs(req struct {
	Protocol string `json:"protocol"`
	DstPort  string `json:"dst_port"`
	SrcIP    string `json:"src_ip"`
	Target   string `json:"target"`
	Comment  string `json:"comment"`
}) []string {
	args := []string{}
	target := req.Target
	if target == "" {
		target = "allow"
	}
	if target == "ACCEPT" {
		target = "allow"
	} else if target == "DROP" {
		target = "deny"
	}
	args = append(args, target)

	if req.SrcIP != "" {
		args = append(args, "from", req.SrcIP)
	}
	if req.DstPort != "" {
		args = append(args, "to", "any", "port", req.DstPort)
	}
	if req.Protocol != "" && req.Protocol != "all" {
		args = append(args, "proto", req.Protocol)
	}
	return args
}

func buildIPTablesAddArgs(req struct {
	Protocol string `json:"protocol"`
	DstPort  string `json:"dst_port"`
	SrcIP    string `json:"src_ip"`
	Target   string `json:"target"`
	Comment  string `json:"comment"`
}) []string {
	args := []string{"-A", "INPUT"}
	if req.Protocol != "" && req.Protocol != "all" {
		args = append(args, "-p", req.Protocol)
	}
	if req.SrcIP != "" {
		args = append(args, "-s", req.SrcIP)
	}
	if req.DstPort != "" {
		args = append(args, "--dport", req.DstPort)
	}
	target := req.Target
	if target == "" {
		target = "ACCEPT"
	}
	args = append(args, "-j", target)
	if req.Comment != "" {
		args = append(args, "-m", "comment", "--comment", req.Comment)
	}
	return args
}

// DeleteFirewallRule 删除防火墙规则
// @Summary 删除防火墙规则
// @Tags 防火墙
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则序号"
// @Success 200 {object} map[string]interface{}
// @Router /security/firewall/rules/{id} [delete]
func DeleteFirewallRule(c *gin.Context) {
	ruleNum := c.Param("id")
	if runtime.GOOS == "windows" {
		fail(c, 500, "Windows 不支持")
		return
	}

	_, err := exec.LookPath("ufw")
	if err == nil {
		out, err := exec.Command("ufw", "delete", ruleNum).CombinedOutput()
		if err != nil {
			fail(c, 500, fmt.Sprintf("删除失败: %s", string(out)))
			return
		}
		success(c, gin.H{"msg": "规则已删除"})
		return
	}

	out, err := exec.Command("iptables", "-D", "INPUT", ruleNum).CombinedOutput()
	if err != nil {
		fail(c, 500, fmt.Sprintf("删除失败: %s", string(out)))
		return
	}
	success(c, gin.H{"msg": "规则已删除"})
}

// GetFirewallPorts 获取已开放端口
// @Summary 获取服务器已监听端口列表
// @Tags 防火墙
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /security/firewall/ports [get]
func GetFirewallPorts(c *gin.Context) {
	out, err := exec.Command("ss", "-tlnp").CombinedOutput()
	if err != nil {
		// fallback to netstat
		out, err = exec.Command("netstat", "-tlnp").CombinedOutput()
		if err != nil {
			fail(c, 500, "获取端口失败")
			return
		}
	}

	type PortInfo struct {
		Proto   string `json:"proto"`
		Local   string `json:"local"`
		Address string `json:"address"`
		Process string `json:"process"`
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var ports []PortInfo
	seen := map[string]bool{}
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		var pi PortInfo
		if strings.HasPrefix(fields[0], "tcp") || strings.HasPrefix(fields[0], "udp") {
			pi.Proto = fields[0]
			pi.Local = fields[3]
		} else {
			continue
		}
		// extract process
		for _, f := range fields {
			if strings.Contains(f, `"`) {
				pi.Process = f
			}
		}
		key := pi.Proto + pi.Local
		if !seen[key] {
			seen[key] = true
			ports = append(ports, pi)
		}
	}
	success(c, ports)
}

// RestartFirewall 重启防火墙
// @Summary 重启防火墙服务
// @Tags 防火墙
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /security/firewall/restart [post]
func RestartFirewall(c *gin.Context) {
	if runtime.GOOS == "windows" {
		fail(c, 500, "Windows 不支持")
		return
	}

	_, err := exec.LookPath("ufw")
	if err == nil {
		out, err := exec.Command("systemctl", "restart", "ufw").CombinedOutput()
		if err != nil {
			// try reload
			out, err = exec.Command("ufw", "reload").CombinedOutput()
			if err != nil {
				fail(c, 500, fmt.Sprintf("重启失败: %s", string(out)))
				return
			}
		}
		success(c, gin.H{"msg": "防火墙已重启"})
		return
	}

	out, err := exec.Command("systemctl", "restart", "iptables").CombinedOutput()
	if err != nil {
		fail(c, 500, fmt.Sprintf("重启失败: %s", string(out)))
		return
	}
	success(c, gin.H{"msg": "防火墙已重启"})
}
