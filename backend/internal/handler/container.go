package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

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

func GetContainer(c *gin.Context) {
	id := c.Param("id")
	var container model.Container
	if err := config.DB.First(&container, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, container)
}

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
