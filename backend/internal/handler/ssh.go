package handler

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var wsSSHUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type CreateSSHAccountReq struct {
	Name        string `json:"name" binding:"required"`
	Host        string `json:"host" binding:"required"`
	Port        int    `json:"port"`
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password"`
	AuthMethod  string `json:"auth_method"`
	PrivateKey  string `json:"private_key"`
	Description string `json:"description"`
}

// ListSSHAccounts 获取 SSH 账号列表
func ListSSHAccounts(c *gin.Context) {
	var accounts []model.SSHAccount
	var total int64
	query := config.DB.Model(&model.SSHAccount{})
	query.Count(&total)
	paginate(c, query).Order("id desc").Find(&accounts)
	for i := range accounts {
		accounts[i].Password = ""
		accounts[i].PrivateKey = ""
	}
	pageResult(c, accounts, total)
}

// CreateSSHAccount 创建 SSH 账号
func CreateSSHAccount(c *gin.Context) {
	var req CreateSSHAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if req.Port == 0 {
		req.Port = 22
	}
	if req.AuthMethod == "" {
		req.AuthMethod = "password"
	}
	account := model.SSHAccount{
		Name: req.Name, Host: req.Host, Port: req.Port,
		Username: req.Username, Password: req.Password,
		AuthMethod: req.AuthMethod, PrivateKey: req.PrivateKey,
		Status: "active", Description: req.Description,
	}
	if err := config.DB.Create(&account).Error; err != nil {
		fail(c, 500, "创建失败")
		return
	}
	addLog("info", "ssh", fmt.Sprintf("创建 SSH 账号: %s (%s@%s:%d)", account.Name, account.Username, account.Host, account.Port))
	account.Password = ""
	account.PrivateKey = ""
	success(c, account)
}

// GetSSHAccount 获取 SSH 账号详情（隐藏敏感字段）
func GetSSHAccount(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	account.Password = ""
	account.PrivateKey = ""
	success(c, account)
}

// GetSSHAccountFull 获取 SSH 账号完整信息
func GetSSHAccountFull(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, account)
}

// UpdateSSHAccount 更新 SSH 账号基本信息
func UpdateSSHAccount(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Host        *string `json:"host"`
		Port        *int    `json:"port"`
		Username    *string `json:"username"`
		AuthMethod  *string `json:"auth_method"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Host != nil {
		updates["host"] = *req.Host
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.AuthMethod != nil {
		updates["auth_method"] = *req.AuthMethod
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if len(updates) > 0 {
		config.DB.Model(&account).Updates(updates)
	}
	config.DB.First(&account, id)
	addLog("info", "ssh", fmt.Sprintf("更新 SSH 账号: %s", account.Name))
	account.Password = ""
	account.PrivateKey = ""
	success(c, account)
}

// ChangeSSHCredential 修改密码或密钥
func ChangeSSHCredential(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	var req struct {
		AuthMethod *string `json:"auth_method"`
		Password   *string `json:"password"`
		PrivateKey *string `json:"private_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	updates := map[string]any{}
	if req.AuthMethod != nil {
		updates["auth_method"] = *req.AuthMethod
	}
	if req.Password != nil {
		updates["password"] = *req.Password
	}
	if req.PrivateKey != nil {
		updates["private_key"] = *req.PrivateKey
	}
	if len(updates) == 0 {
		fail(c, 400, "未提供修改内容")
		return
	}
	config.DB.Model(&account).Updates(updates)
	config.DB.First(&account, id)
	addLog("info", "ssh", fmt.Sprintf("修改 SSH 凭证: %s (%s)", account.Name, account.AuthMethod))
	account.Password = ""
	account.PrivateKey = ""
	success(c, account)
}

// DeleteSSHAccount 删除 SSH 账号
func DeleteSSHAccount(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	if err := config.DB.Delete(&model.SSHAccount{}, id).Error; err != nil {
		fail(c, 500, "删除失败")
		return
	}
	addLog("info", "ssh", fmt.Sprintf("删除 SSH 账号: %s (%s@%s:%d)", account.Name, account.Username, account.Host, account.Port))
	success(c, nil)
}

// TestSSHConnection 真实 SSH 认证测试
func TestSSHConnection(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	client, err := dialSSH(account)
	if err != nil {
		addLog("warn", "ssh", fmt.Sprintf("SSH 测试失败: %s (%s@%s:%d) - %v", account.Name, account.Username, account.Host, account.Port, err))
		success(c, map[string]any{
			"host": account.Host, "port": account.Port,
			"status": "failed", "msg": fmt.Sprintf("认证失败: %v", err),
		})
		return
	}
	defer client.Close()

	remoteInfo := ""
	session, err := client.NewSession()
	if err == nil {
		defer session.Close()
		var buf strings.Builder
		session.Stdout = &buf
		session.Run("uname -a && whoami && uptime")
		remoteInfo = strings.TrimSpace(buf.String())
	}

	addLog("info", "ssh", fmt.Sprintf("SSH 测试成功: %s (%s@%s:%d)", account.Name, account.Username, account.Host, account.Port))
	success(c, map[string]any{
		"host": account.Host, "port": account.Port,
		"status": "connected", "msg": "认证成功",
		"remote_info": remoteInfo,
	})
}

// GenerateSSHKeyPair 生成 SSH 密钥对
func GenerateSSHKeyPair(c *gin.Context) {
	curve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		fail(c, 500, "密钥生成失败")
		return
	}

	derBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		fail(c, 500, "私钥编码失败")
		return
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derBytes})

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		fail(c, 500, "公钥生成失败")
		return
	}
	publicKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey)))

	addLog("info", "ssh", "生成 SSH 密钥对")
	success(c, map[string]any{
		"private_key": string(privatePEM),
		"public_key":  publicKeyStr,
	})
}

// InstallSSHKey 一键安装密钥（用密码登录部署公钥，然后切换为密钥认证）
func InstallSSHKey(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	// 生成新密钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fail(c, 500, "密钥生成失败")
		return
	}
	derBytes, _ := x509.MarshalECPrivateKey(privateKey)
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derBytes})
	publicKey, _ := ssh.NewPublicKey(&privateKey.PublicKey)
	publicKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey)))

	// 用密码连接部署公钥
	if account.Password == "" {
		fail(c, 400, "需要密码认证才能安装密钥")
		return
	}
	account.AuthMethod = "password"
	client, err := dialSSH(account)
	if err != nil {
		fail(c, 500, fmt.Sprintf("SSH 连接失败: %v", err))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fail(c, 500, "创建会话失败")
		return
	}
	defer session.Close()

	cmd := fmt.Sprintf(`mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo '%s' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && echo OK`, publicKeyStr)
	var out strings.Builder
	session.Stdout = &out
	if err := session.Run(cmd); err != nil || !strings.Contains(out.String(), "OK") {
		fail(c, 500, fmt.Sprintf("部署公钥失败: %v", err))
		return
	}

	config.DB.Model(&account).Updates(map[string]any{
		"auth_method": "key",
		"private_key": string(privatePEM),
		"public_key":  publicKeyStr,
	})

	addLog("info", "ssh", fmt.Sprintf("一键安装密钥: %s (%s@%s)", account.Name, account.Username, account.Host))
	success(c, map[string]any{
		"msg":        "密钥安装成功，已切换为密钥认证",
		"public_key": publicKeyStr,
	})
}

// ChangeRemotePassword 修改远程服务器用户密码
func ChangeRemotePassword(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	var req struct {
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	client, err := dialSSH(account)
	if err != nil {
		fail(c, 500, fmt.Sprintf("SSH 连接失败: %v", err))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fail(c, 500, "创建会话失败")
		return
	}
	defer session.Close()

	cmd := fmt.Sprintf(`echo '%s:%s' | sudo chpasswd`, account.Username, req.NewPassword)
	session.Run(cmd)

	config.DB.Model(&account).Update("password", req.NewPassword)
	addLog("info", "ssh", fmt.Sprintf("修改远程密码: %s (%s@%s)", account.Name, account.Username, account.Host))
	success(c, map[string]any{"msg": "密码修改成功"})
}

// ChangeSSHPort 修改远程 SSH 端口
func ChangeSSHPort(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	var req struct {
		NewPort int `json:"new_port" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	if req.NewPort < 1 || req.NewPort > 65535 {
		fail(c, 400, "端口范围 1-65535")
		return
	}

	client, err := dialSSH(account)
	if err != nil {
		fail(c, 500, fmt.Sprintf("SSH 连接失败: %v", err))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fail(c, 500, "创建会话失败")
		return
	}
	defer session.Close()

	cmd := fmt.Sprintf(`sudo sed -i 's/^#\?Port .*/Port %d/' /etc/ssh/sshd_config && sudo systemctl restart sshd || sudo service ssh restart`, req.NewPort)
	if err := session.Run(cmd); err != nil {
		fail(c, 500, fmt.Sprintf("修改远程端口失败: %v", err))
		return
	}

	config.DB.Model(&account).Update("port", req.NewPort)
	config.DB.First(&account, id)
	addLog("info", "ssh", fmt.Sprintf("修改 SSH 端口: %s -> %d", account.Name, req.NewPort))
	account.Password = ""
	account.PrivateKey = ""
	success(c, account)
}

// RestartSSHD 重启远程 SSH 服务
func RestartSSHD(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	client, err := dialSSH(account)
	if err != nil {
		fail(c, 500, fmt.Sprintf("SSH 连接失败: %v", err))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fail(c, 500, "创建会话失败")
		return
	}
	defer session.Close()

	session.Run("sudo systemctl restart sshd || sudo service ssh restart")

	addLog("info", "ssh", fmt.Sprintf("重启 SSH 服务: %s (%s@%s)", account.Name, account.Username, account.Host))
	success(c, map[string]any{"msg": "SSH 服务已重启"})
}

// GetSSHdConfig 获取远程 sshd_config
func GetSSHdConfig(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	client, err := dialSSH(account)
	if err != nil {
		fail(c, 500, fmt.Sprintf("SSH 连接失败: %v", err))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fail(c, 500, "创建会话失败")
		return
	}
	defer session.Close()

	var stdout strings.Builder
	session.Stdout = &stdout
	session.Run("cat /etc/ssh/sshd_config 2>/dev/null")

	success(c, map[string]any{"content": stdout.String()})
}

// SaveSSHdConfig 保存远程 sshd_config
func SaveSSHdConfig(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	client, err := dialSSH(account)
	if err != nil {
		fail(c, 500, fmt.Sprintf("SSH 连接失败: %v", err))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fail(c, 500, "创建会话失败")
		return
	}
	defer session.Close()

	cmd := fmt.Sprintf(`sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak.$(date +%%%%Y%%%%m%%%%d%%%%H%%%%M%%%%S) && echo '%s' | sudo tee /etc/ssh/sshd_config > /dev/null`, req.Content)
	session.Run(cmd)

	addLog("info", "ssh", fmt.Sprintf("更新 sshd_config: %s", account.Name))
	success(c, map[string]any{"msg": "配置已保存（已自动备份）"})
}

// ExecuteSSHCommand 执行单条远程命令
func ExecuteSSHCommand(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	var req struct {
		Command string `json:"command" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	client, err := dialSSH(account)
	if err != nil {
		fail(c, 500, fmt.Sprintf("SSH 连接失败: %v", err))
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fail(c, 500, "创建会话失败")
		return
	}
	defer session.Close()

	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run(req.Command)

	addLog("info", "ssh", fmt.Sprintf("执行命令 [%s]: %s", account.Name, req.Command))
	success(c, map[string]any{
		"stdout":  stdout.String(),
		"stderr":  stderr.String(),
		"exit_ok": err == nil,
	})
}

// WebSSHHandler WebSocket SSH 终端
func WebSSHHandler(c *gin.Context) {
	id := c.Param("id")
	var account model.SSHAccount
	if err := config.DB.First(&account, id).Error; err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "未找到"})
		return
	}

	ws, err := wsSSHUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	client, err := dialSSH(account)
	if err != nil {
		ws.WriteJSON(gin.H{"error": fmt.Sprintf("SSH 连接失败: %v", err)})
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		ws.WriteJSON(gin.H{"error": "创建会话失败"})
		return
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 80, 40, modes); err != nil {
		ws.WriteJSON(gin.H{"error": "PTY 请求失败"})
		return
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return
	}

	if err := session.Shell(); err != nil {
		ws.WriteJSON(gin.H{"error": "启动 Shell 失败"})
		return
	}

	addLog("info", "ssh", fmt.Sprintf("WebSSH 终端连接: %s (%s@%s:%d)", account.Name, account.Username, account.Host, account.Port))

	done := make(chan struct{}, 2)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				ws.WriteMessage(websocket.TextMessage, buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				break
			}
			msgStr := string(msg)
			if strings.HasPrefix(msgStr, "__resize__:") {
				parts := strings.Split(strings.TrimPrefix(msgStr, "__resize__:"), "x")
				if len(parts) == 2 {
					var w, h int
					fmt.Sscanf(parts[0], "%d", &w)
					fmt.Sscanf(parts[1], "%d", &h)
					if w > 0 && h > 0 {
						session.WindowChange(h, w)
					}
				}
				continue
			}
			stdin.Write(msg)
		}
		done <- struct{}{}
	}()

	<-done
	addLog("info", "ssh", fmt.Sprintf("WebSSH 终端断开: %s", account.Name))
}

// dialSSH 建立 SSH 连接（支持密码和密钥认证，自动回退）
func dialSSH(account model.SSHAccount) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if account.AuthMethod == "key" && account.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(account.PrivateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	if account.Password != "" {
		authMethods = append(authMethods, ssh.Password(account.Password))
	}
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("未配置认证方式（密码或密钥）")
	}

	cfg := &ssh.ClientConfig{
		User:            account.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", account.Host, account.Port)
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil && account.AuthMethod == "key" && account.Password != "" {
		// 密钥失败，回退密码
		fallback := &ssh.ClientConfig{
			User:            account.Username,
			Auth:            []ssh.AuthMethod{ssh.Password(account.Password)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		}
		return ssh.Dial("tcp", addr, fallback)
	}
	return client, err
}

// 检查端口是否可用
func checkLocalPort(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
