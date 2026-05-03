package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

// ListContainers 获取容器列表
// @Summary 获取 Docker 容器列表（分页）
// @Tags 容器管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /containers [get]
func ListContainers(c *gin.Context) {
	var containers []model.Container
	var total int64
	query := config.DB.Model(&model.Container{})
	query.Count(&total)
	paginate(c, query).Order("id desc").Find(&containers)
	pageResult(c, containers, total)
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
