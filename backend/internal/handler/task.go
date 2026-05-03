package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"
	"opsmanage/internal/scheduler"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ListTasks 获取任务列表
// @Summary 获取计划任务列表（分页）
// @Tags 计划任务
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /tasks [get]
func ListTasks(c *gin.Context) {
	var tasks []model.Task
	var total int64
	query := config.DB.Model(&model.Task{})
	query.Count(&total)
	paginate(c, query).Order("id desc").Find(&tasks)
	pageResult(c, tasks, total)
}

type CreateTaskReq struct {
	Name     string `json:"name" binding:"required"`
	Command  string `json:"command" binding:"required"`
	CronExpr string `json:"cron_expr" binding:"required"`
}

// CreateTask 创建计划任务
// @Summary 创建计划任务
// @Tags 计划任务
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateTaskReq true "任务信息"
// @Success 200 {object} map[string]interface{}
// @Router /tasks [post]
func CreateTask(c *gin.Context) {
	var req CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	task := model.Task{
		Name:     req.Name,
		Command:  req.Command,
		CronExpr: req.CronExpr,
		Status:   "enabled",
	}

	if err := config.DB.Create(&task).Error; err != nil {
		fail(c, 500, "创建失败")
		return
	}
	scheduler.AddTask(task)
	addLog("info", "task", "创建计划任务: "+task.Name)
	success(c, task)
}

// GetTask 获取任务详情
// @Summary 获取计划任务详情
// @Tags 计划任务
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /tasks/{id} [get]
func GetTask(c *gin.Context) {
	id := c.Param("id")
	var task model.Task
	if err := config.DB.First(&task, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, task)
}

// UpdateTask 更新计划任务
// @Summary 更新计划任务
// @Tags 计划任务
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /tasks/{id} [put]
func UpdateTask(c *gin.Context) {
	id := c.Param("id")
	var task model.Task
	if err := config.DB.First(&task, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var req struct {
		Name     *string `json:"name"`
		Command  *string `json:"command"`
		CronExpr *string `json:"cron_expr"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	updates := map[string]any{}
	if req.Name != nil { updates["name"] = *req.Name }
	if req.Command != nil { updates["command"] = *req.Command }
	if req.CronExpr != nil { updates["cron_expr"] = *req.CronExpr }

	if len(updates) > 0 {
		config.DB.Model(&task).Updates(updates)
	}
	config.DB.First(&task, id)
	scheduler.ReloadTask(task)
	success(c, task)
}

// DeleteTask 删除计划任务
// @Summary 删除计划任务
// @Tags 计划任务
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /tasks/{id} [delete]
func DeleteTask(c *gin.Context) {
	id := c.Param("id")
	var task model.Task
	config.DB.First(&task, id)
	scheduler.RemoveTask(task.ID)
	if err := config.DB.Delete(&model.Task{}, id).Error; err != nil {
		fail(c, 500, "删除失败")
		return
	}
	success(c, nil)
}

// RunTask 手动执行任务
// @Summary 手动触发执行计划任务
// @Tags 计划任务
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /tasks/{id}/run [post]
func RunTask(c *gin.Context) {
	id := c.Param("id")
	var task model.Task
	if err := config.DB.First(&task, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	go func() {
		out, err := exec.Command("bash", "-c", task.Command).CombinedOutput()
		result := "success"
		if err != nil {
			result = "failed"
		}
		config.DB.Model(&task).Updates(map[string]any{
			"last_run":    time.Now(),
			"last_result": result,
			"last_output": strings.TrimSpace(string(out)),
		})
	}()

	success(c, gin.H{"msg": "任务已触发执行"})
}

// ToggleTask 切换任务状态
// @Summary 启用/禁用计划任务
// @Tags 计划任务
// @Produce json
// @Security BearerAuth
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /tasks/{id}/toggle [post]
func ToggleTask(c *gin.Context) {
	id := c.Param("id")
	var task model.Task
	if err := config.DB.First(&task, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	if task.Status == "enabled" {
		config.DB.Model(&task).Update("status", "disabled")
	} else {
		config.DB.Model(&task).Update("status", "enabled")
	}
	config.DB.First(&task, id)
	scheduler.ReloadTask(task)
	success(c, task)
}
