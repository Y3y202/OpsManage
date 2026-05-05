package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

// ListContainers 获取容器列表
func ListContainers(c *gin.Context) {
	// 从 Docker 直接获取真实容器列表
	out, err := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}").CombinedOutput()
	if err != nil {
		// Docker 不可用时，回退到数据库
		var containers []model.Container
		var total int64
		query := config.DB.Model(&model.Container{})
		query.Count(&total)
		paginate(c, query).Order("id desc").Find(&containers)
		pageResult(c, containers, total)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var containers []map[string]string
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 4 {
			continue
		}
		container := map[string]string{
			"container_id": parts[0],
			"name":         parts[1],
			"image":        parts[2],
			"status":       parts[3],
		}
		if len(parts) > 4 {
			container["ports"] = parts[4]
		}
		// 解析状态
		if strings.Contains(parts[3], "Up") {
			container["status"] = "running"
		} else {
			container["status"] = "stopped"
		}
		containers = append(containers, container)
	}

	success(c, gin.H{"list": containers, "total": len(containers)})
}

type CreateContainerReq struct {
	Name    string `json:"name" binding:"required"`
	Image   string `json:"image" binding:"required"`
	Ports   string `json:"ports"`
	Volumes string `json:"volumes"`
	Env     string `json:"env"`
}

// CreateContainer 创建容器
// @Summary 创建 Docker 容器
// @Tags 容器管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateContainerReq true "容器信息"
// @Success 200 {object} map[string]interface{}
// @Router /containers [post]
func CreateContainer(c *gin.Context) {
	var req CreateContainerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	container := model.Container{
		Name:      req.Name,
		Image:     req.Image,
		Ports:     req.Ports,
		Volumes:   req.Volumes,
		Env:       req.Env,
		Status:    "created",
	}

	if err := config.DB.Create(&container).Error; err != nil {
		fail(c, 500, "创建失败: "+err.Error())
		return
	}

	addLog("info", "container", "创建容器: "+container.Name+" (镜像: "+container.Image+")")
	success(c, container)
}

// GetContainer 获取容器详情
// @Summary 获取容器详情
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "容器ID"
// @Success 200 {object} map[string]interface{}
// @Router /containers/{id} [get]
func GetContainer(c *gin.Context) {
	id := c.Param("id")
	var container model.Container
	if err := config.DB.First(&container, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, container)
}

// DeleteContainer 删除容器
// @Summary 删除容器（docker rm -f）
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "容器ID"
// @Success 200 {object} map[string]interface{}
// @Router /containers/{id} [delete]
func DeleteContainer(c *gin.Context) {
	id := c.Param("id")
	var container model.Container
	if err := config.DB.First(&container, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	if container.ContainerID != "" {
		exec.Command("docker", "rm", "-f", container.ContainerID).Run()
	}
	config.DB.Delete(&container)
	success(c, nil)
}

// StartContainer 启动容器
// @Summary 启动 Docker 容器
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "容器ID"
// @Success 200 {object} map[string]interface{}
// @Router /containers/{id}/start [post]
func StartContainer(c *gin.Context) {
	id := c.Param("id")
	var container model.Container
	if err := config.DB.First(&container, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	out, err := exec.Command("docker", "start", container.ContainerID).CombinedOutput()
	if err != nil {
		fail(c, 500, "启动失败: "+string(out))
		return
	}
	config.DB.Model(&container).Update("status", "running")
	success(c, gin.H{"status": "running"})
}

// StopContainer 停止容器
// @Summary 停止 Docker 容器
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "容器ID"
// @Success 200 {object} map[string]interface{}
// @Router /containers/{id}/stop [post]
func StopContainer(c *gin.Context) {
	id := c.Param("id")
	var container model.Container
	if err := config.DB.First(&container, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	out, err := exec.Command("docker", "stop", container.ContainerID).CombinedOutput()
	if err != nil {
		fail(c, 500, "停止失败: "+string(out))
		return
	}
	config.DB.Model(&container).Update("status", "stopped")
	success(c, gin.H{"status": "stopped"})
}

// RestartContainer 重启容器
// @Summary 重启 Docker 容器
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "容器ID"
// @Success 200 {object} map[string]interface{}
// @Router /containers/{id}/restart [post]
func RestartContainer(c *gin.Context) {
	id := c.Param("id")
	var container model.Container
	if err := config.DB.First(&container, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	out, err := exec.Command("docker", "restart", container.ContainerID).CombinedOutput()
	if err != nil {
		fail(c, 500, "重启失败: "+string(out))
		return
	}
	config.DB.Model(&container).Update("status", "running")
	success(c, gin.H{"status": "running"})
}

// PullImage 拉取镜像
// @Summary 拉取 Docker 镜像
// @Tags 容器管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object true "镜像名" {"image":"string"}
// @Success 200 {object} map[string]interface{}
// @Router /containers/images/pull [post]
func PullImage(c *gin.Context) {
	var req struct {
		Image string `json:"image" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	out, err := exec.Command("docker", "pull", req.Image).CombinedOutput()
	if err != nil {
		fail(c, 500, "拉取失败: "+string(out))
		return
	}
	success(c, gin.H{"output": strings.TrimSpace(string(out))})
}

// ListImages 获取镜像列表
// @Summary 获取 Docker 镜像列表
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /containers/images [get]
func ListImages(c *gin.Context) {
	out, err := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}\t{{.ID}}\t{{.Size}}").CombinedOutput()
	if err != nil {
		fail(c, 500, "获取镜像列表失败")
		return
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var images []map[string]string
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		images = append(images, map[string]string{
			"name":  parts[0],
			"id":    parts[1],
			"size":  parts[2],
		})
	}
	success(c, images)
}

// GetContainerLogs 获取容器日志
// @Summary 获取容器日志（最近100行）
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "容器ID"
// @Success 200 {object} map[string]interface{}
// @Router /containers/{id}/logs [get]
func GetContainerLogs(c *gin.Context) {
	id := c.Param("id")
	var container model.Container
	if err := config.DB.First(&container, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	out, err := exec.Command("docker", "logs", "--tail", "100", container.ContainerID).CombinedOutput()
	if err != nil {
		fail(c, 500, "获取日志失败")
		return
	}
	success(c, gin.H{"logs": string(out)})
}

// GetDockerOverview 获取 Docker 总览概要
func GetDockerOverview(c *gin.Context) {
	// 从 Docker 获取真实容器统计
	runningCount := 0
	totalCount := 0

	out, err := exec.Command("docker", "ps", "-a", "--format", "{{.Status}}").CombinedOutput()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			totalCount++
			if strings.Contains(line, "Up") {
				runningCount++
			}
		}
	}

	// 从 Docker 获取真实数据
	overview := gin.H{
		"containers_total":   totalCount,
		"containers_running": runningCount,
		"images":             countDockerItems("images", "-q"),
		"networks":           countDockerItems("network", "ls", "-q"),
		"volumes":            countDockerItems("volume", "ls", "-q"),
		"compose":            0,
		"compose_templates":  0,
		"registries":         1,
	}

	// 磁盘占用
	overview["disk_usage"] = getDockerDiskUsage()
	overview["socket"] = "unix:///var/run/docker.sock"

	success(c, overview)
}

func countDockerItems(args ...string) int {
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func getDockerDiskUsage() map[string]any {
	out, err := exec.Command("docker", "system", "df", "--format", "{{.Type}}\t{{.TotalCount}}\t{{.Size}}\t{{.Reclaimable}}").CombinedOutput()
	if err != nil {
		return map[string]any{}
	}
	disk := map[string]any{}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		switch parts[0] {
		case "Images":
			disk["images"] = map[string]string{"count": parts[1], "size": parts[2], "reclaimable": parts[3]}
		case "Containers":
			disk["containers"] = map[string]string{"count": parts[1], "size": parts[2], "reclaimable": parts[3]}
		case "Local Volumes":
			disk["volumes"] = map[string]string{"count": parts[1], "size": parts[2], "reclaimable": parts[3]}
		case "Build Cache":
			disk["build_cache"] = map[string]string{"count": parts[1], "size": parts[2], "reclaimable": parts[3]}
		}
	}
	return disk
}

// ListDockerNetworks 获取 Docker 网络列表
// @Summary 获取 Docker 网络列表
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /containers/networks [get]
func ListDockerNetworks(c *gin.Context) {
	out, err := exec.Command("docker", "network", "ls", "--format", "{{.ID}}\t{{.Name}}\t{{.Driver}}\t{{.Scope}}").CombinedOutput()
	if err != nil {
		fail(c, 500, "获取网络列表失败")
		return
	}
	var networks []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}
		networks = append(networks, map[string]string{
			"id": parts[0], "name": parts[1], "driver": parts[2], "scope": parts[3],
		})
	}
	success(c, networks)
}

// ListDockerVolumes 获取 Docker 存储卷列表
// @Summary 获取 Docker 存储卷列表
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /containers/volumes [get]
func ListDockerVolumes(c *gin.Context) {
	out, err := exec.Command("docker", "volume", "ls", "--format", "{{.Name}}\t{{.Driver}}\t{{.Mountpoint}}").CombinedOutput()
	if err != nil {
		fail(c, 500, "获取存储卷列表失败")
		return
	}
	var volumes []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		volumes = append(volumes, map[string]string{
			"name": parts[0], "driver": parts[1], "mountpoint": parts[2],
		})
	}
	success(c, volumes)
}

// RemoveImage 删除镜像
func RemoveImage(c *gin.Context) {
	id := c.Param("id")
	out, err := exec.Command("docker", "rmi", "-f", id).CombinedOutput()
	if err != nil {
		fail(c, 500, "删除镜像失败: "+string(out))
		return
	}
	success(c, gin.H{"msg": "镜像已删除"})
}

// RemoveNetwork 删除网络
func RemoveNetwork(c *gin.Context) {
	id := c.Param("id")
	out, err := exec.Command("docker", "network", "rm", id).CombinedOutput()
	if err != nil {
		fail(c, 500, "删除网络失败: "+string(out))
		return
	}
	success(c, gin.H{"msg": "网络已删除"})
}

// RemoveVolume 删除存储卷
func RemoveVolume(c *gin.Context) {
	id := c.Param("id")
	out, err := exec.Command("docker", "volume", "rm", "-f", id).CombinedOutput()
	if err != nil {
		fail(c, 500, "删除存储卷失败: "+string(out))
		return
	}
	success(c, gin.H{"msg": "存储卷已删除"})
}

// PruneDocker 清理 Docker
func PruneDocker(c *gin.Context) {
	typ := c.DefaultQuery("type", "all")
	var args []string
	switch typ {
	case "images":
		args = []string{"image", "prune", "-f"}
	case "containers":
		args = []string{"container", "prune", "-f"}
	case "volumes":
		args = []string{"volume", "prune", "-f"}
	default:
		args = []string{"system", "prune", "-f"}
	}
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		fail(c, 500, "清理失败: "+string(out))
		return
	}
	success(c, gin.H{"msg": "清理完成", "output": strings.TrimSpace(string(out))})
}

// ========== 镜像仓库 CRUD ==========
func ListRegistries(c *gin.Context) {
	var list []model.DockerRegistry
	config.DB.Find(&list)
	success(c, list)
}
func CreateRegistry(c *gin.Context) {
	var reg model.DockerRegistry
	if err := c.ShouldBindJSON(&reg); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	config.DB.Create(&reg)
	success(c, reg)
}
func DeleteRegistry(c *gin.Context) {
	id := c.Param("id")
	config.DB.Delete(&model.DockerRegistry{}, id)
	success(c, nil)
}

// ========== 编排项目 CRUD ==========
func ListComposeProjects(c *gin.Context) {
	var list []model.ComposeProject
	config.DB.Find(&list)
	success(c, list)
}
func CreateComposeProject(c *gin.Context) {
	var proj model.ComposeProject
	if err := c.ShouldBindJSON(&proj); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	config.DB.Create(&proj)
	success(c, proj)
}
func DeleteComposeProject(c *gin.Context) {
	id := c.Param("id")
	config.DB.Delete(&model.ComposeProject{}, id)
	success(c, nil)
}
func StartComposeProject(c *gin.Context) {
	id := c.Param("id")
	var proj model.ComposeProject
	if err := config.DB.First(&proj, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	out, err := exec.Command("docker", "compose", "-f", proj.Path, "up", "-d").CombinedOutput()
	if err != nil {
		fail(c, 500, "启动失败: "+string(out))
		return
	}
	config.DB.Model(&proj).Update("status", "running")
	success(c, gin.H{"msg": "已启动"})
}
func StopComposeProject(c *gin.Context) {
	id := c.Param("id")
	var proj model.ComposeProject
	if err := config.DB.First(&proj, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	out, err := exec.Command("docker", "compose", "-f", proj.Path, "down").CombinedOutput()
	if err != nil {
		fail(c, 500, "停止失败: "+string(out))
		return
	}
	config.DB.Model(&proj).Update("status", "stopped")
	success(c, gin.H{"msg": "已停止"})
}

// ========== 编排模板 CRUD ==========
func ListComposeTemplates(c *gin.Context) {
	var list []model.ComposeTemplate
	config.DB.Find(&list)
	success(c, list)
}
func CreateComposeTemplate(c *gin.Context) {
	var tpl model.ComposeTemplate
	if err := c.ShouldBindJSON(&tpl); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	config.DB.Create(&tpl)
	success(c, tpl)
}
func DeleteComposeTemplate(c *gin.Context) {
	id := c.Param("id")
	config.DB.Delete(&model.ComposeTemplate{}, id)
	success(c, nil)
}
