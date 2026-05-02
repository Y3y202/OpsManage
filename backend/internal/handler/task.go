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

func GetTask(c *gin.Context) {
	id := c.Param("id")
	var task model.Task
	if err := config.DB.First(&task, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, task)
}

func UpdateTask(c *gin.Context) {
	id := c.Param("id")
	var task model.Task
	if err := config.DB.First(&task, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var updates map[string]any
	if err := c.ShouldBindJSON(&updates); err != nil {
		fail(c, 400, "参数错误")
		return
	}
	delete(updates, "id")
	delete(updates, "created_at")
	config.DB.Model(&task).Updates(updates)
	config.DB.First(&task, id)
	scheduler.ReloadTask(task)
	success(c, task)
}

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
