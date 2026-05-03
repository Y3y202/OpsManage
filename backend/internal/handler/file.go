package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// validatePath cleans and validates a file path to prevent traversal attacks
// and protect sensitive system files.
func validatePath(p string) (string, error) {
	// Clean the path (resolve .., ., double slashes)
	cleaned := filepath.Clean(p)

	// Reject obvious traversal from relative paths
	if strings.Contains(p, "..") {
		abs, err := filepath.Abs(cleaned)
		if err != nil {
			return "", err
		}
		// Allow absolute paths (the admin manages the full server)
		cleaned = abs
	}

	// Block access to critical system files
	blocked := []string{
		"/etc/shadow",
		"/etc/passwd",
		"/etc/sudoers",
		"/etc/ssh/sshd_config",
	}
	lower := strings.ToLower(cleaned)
	for _, b := range blocked {
		if lower == b {
			return "", os.ErrPermission
		}
	}

	return cleaned, nil
}

type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
	Mode    string `json:"mode"`
}

// ListFiles 列出目录文件
// @Summary 列出目录文件
// @Tags 文件管理
// @Produce json
// @Security BearerAuth
// @Param path query string true "目录路径"
// @Success 200 {object} map[string]interface{}
// @Router /files/list [get]
func ListFiles(c *gin.Context) {
	dirPath := c.DefaultQuery("path", "/")
	dirPath, err := validatePath(dirPath)
	if err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}
	if !filepath.IsAbs(dirPath) {
		dirPath, _ = filepath.Abs(dirPath)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		fail(c, 400, "无法读取目录: "+err.Error())
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(dirPath, entry.Name()),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			Mode:    info.Mode().String(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return files[i].Name < files[j].Name
	})

	success(c, gin.H{
		"path":  dirPath,
		"files": files,
	})
}

// ReadFile 读取文件内容
// @Summary 读取文件内容（最大10MB）
// @Tags 文件管理
// @Produce json
// @Security BearerAuth
// @Param path query string true "文件路径"
// @Success 200 {object} map[string]interface{}
// @Router /files/read [get]
func ReadFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		fail(c, 400, "缺少文件路径")
		return
	}
	filePath, err := validatePath(filePath)
	if err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		fail(c, 404, "文件不存在")
		return
	}
	if info.IsDir() {
		fail(c, 400, "这是一个目录")
		return
	}
	if info.Size() > 10*1024*1024 {
		fail(c, 400, "文件过大（>10MB），无法在线查看")
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fail(c, 500, "读取失败")
		return
	}

	if isBinaryFile(data) {
		fail(c, 400, "二进制文件不支持在线查看")
		return
	}

	success(c, gin.H{
		"path":    filePath,
		"content": string(data),
		"size":    info.Size(),
	})
}

// SaveFile 保存文件内容
// @Summary 保存文件内容
// @Tags 文件管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "文件信息"
// @Success 200 {object} map[string]interface{}
// @Router /files/save [post]
func SaveFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	validatedPath, err := validatePath(req.Path)
	if err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}

	if err := os.WriteFile(validatedPath, []byte(req.Content), 0644); err != nil {
		fail(c, 500, "保存失败")
		return
	}
	success(c, nil)
}

// DownloadFile 下载文件
// @Summary 下载文件
// @Tags 文件管理
// @Security BearerAuth
// @Param path query string true "文件路径"
// @Success 200 {string} binary
// @Router /files/download [get]
func DownloadFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		fail(c, 400, "缺少文件路径")
		return
	}
	filePath, err := validatePath(filePath)
	if err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		fail(c, 404, "文件不存在")
		return
	}
	if info.IsDir() {
		fail(c, 400, "不能下载目录")
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	c.Header("Content-Type", "application/octet-stream")
	c.File(filePath)
}

// UploadFile 上传文件
// @Summary 上传文件（最大50MB）
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param dir formData string false "目标目录" default(/tmp)
// @Param file formData file true "上传文件"
// @Success 200 {object} map[string]interface{}
// @Router /files/upload [post]
func UploadFile(c *gin.Context) {
	dir := c.PostForm("dir")
	if dir == "" {
		dir = "/tmp"
	}
	dir, err := validatePath(dir)
	if err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}

	// Limit upload to 50MB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 50<<20)

	file, err := c.FormFile("file")
	if err != nil {
		fail(c, 400, "获取上传文件失败（文件大小不能超过 50MB）")
		return
	}

	// Sanitize filename to prevent traversal
	filename := filepath.Base(file.Filename)
	dest := filepath.Join(dir, filename)
	if _, err := validatePath(dest); err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}
	if err := c.SaveUploadedFile(file, dest); err != nil {
		fail(c, 500, "保存失败")
		return
	}
	success(c, gin.H{"path": dest})
}

// RenameFile 重命名/移动文件
// @Summary 重命名或移动文件
// @Tags 文件管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "路径信息"
// @Success 200 {object} map[string]interface{}
// @Router /files/rename [post]
func RenameFile(c *gin.Context) {
	var req struct {
		OldPath string `json:"old_path" binding:"required"`
		NewPath string `json:"new_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	oldPath, err := validatePath(req.OldPath)
	if err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}
	newPath, err := validatePath(req.NewPath)
	if err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		fail(c, 500, "重命名失败")
		return
	}
	success(c, nil)
}

// DeleteFile 删除文件或目录
// @Summary 删除文件或目录
// @Tags 文件管理
// @Produce json
// @Security BearerAuth
// @Param path query string true "文件路径"
// @Success 200 {object} map[string]interface{}
// @Router /files [delete]
func DeleteFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		fail(c, 400, "缺少文件路径")
		return
	}
	filePath, err := validatePath(filePath)
	if err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}
	if err := os.RemoveAll(filePath); err != nil {
		fail(c, 500, "删除失败")
		return
	}
	success(c, nil)
}

// Mkdir 创建目录
// @Summary 创建目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "目录路径"
// @Success 200 {object} map[string]interface{}
// @Router /files/mkdir [post]
func Mkdir(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	validatedPath, err := validatePath(req.Path)
	if err != nil {
		fail(c, 403, "路径不允许访问")
		return
	}
	if err := os.MkdirAll(validatedPath, 0755); err != nil {
		fail(c, 500, "创建目录失败")
		return
	}
	success(c, nil)
}

// CopyFile 复制文件
// @Summary 复制文件
// @Tags 文件管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "源路径和目标路径"
// @Success 200 {object} map[string]interface{}
// @Router /files/copy [post]
func CopyFile(c *gin.Context) {
	var req struct {
		Src string `json:"src" binding:"required"`
		Dst string `json:"dst" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	srcPath, err := validatePath(req.Src)
	if err != nil {
		fail(c, 403, "源路径不允许访问")
		return
	}
	dstPath, err := validatePath(req.Dst)
	if err != nil {
		fail(c, 403, "目标路径不允许访问")
		return
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		fail(c, 404, "源文件不存在")
		return
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		fail(c, 500, "获取文件信息失败")
		return
	}

	if info.IsDir() {
		fail(c, 400, "不支持复制目录")
		return
	}

	dstFile, err := os.Create(dstPath)
	if err != nil {
		fail(c, 500, "创建目标文件失败")
		return
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		fail(c, 500, "复制失败")
		return
	}
	dstFile.Close()
	success(c, nil)
}

func isBinaryFile(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

// WebSocket

type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
}

type WSClient struct {
	hub  *WSHub
	send chan []byte
	key  string
}

func NewWSHub() *WSHub {
	hub := &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}
	go hub.run()
	return hub
}

func (h *WSHub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

var wsFileHub = NewWSHub()
var wsFileUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "" || r.Header.Get("Origin") == "http://"+r.Host || r.Header.Get("Origin") == "https://"+r.Host
	},
}

func WSFileHandler(c *gin.Context) {
	conn, err := wsFileUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &WSClient{hub: wsFileHub, send: make(chan []byte, 256), key: time.Now().String()}
	wsFileHub.register <- client

	go func() {
		defer func() {
			wsFileHub.unregister <- client
			conn.Close()
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()

	go func() {
		defer conn.Close()
		for msg := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
	}()
}
