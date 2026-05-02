package scheduler

import (
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"opsmanage/internal/config"
	"opsmanage/internal/model"
)

var (
	cronInst *cron.Cron
	mu       sync.Mutex
	entryMap = make(map[uint]cron.EntryID) // task ID -> cron entry ID
)

// Init loads all enabled tasks from DB and starts the cron scheduler.
func Init() {
	cronInst = cron.New(cron.WithSeconds())

	var tasks []model.Task
	config.DB.Where("status = ?", "enabled").Find(&tasks)
	for _, task := range tasks {
		registerTask(task)
	}

	cronInst.Start()
	log.Printf("定时任务调度器已启动，加载 %d 个任务", len(entryMap))
}

// registerTask adds a single task to the cron scheduler. Caller must hold mu.
func registerTask(task model.Task) cron.EntryID {
	id, err := cronInst.AddFunc(task.CronExpr, func() {
		runTask(task.ID)
	})
	if err != nil {
		log.Printf("任务 [%d] %s cron 表达式无效: %v", task.ID, task.Name, err)
		return 0
	}
	entryMap[task.ID] = id

	// Update next_run in DB
	if entry := cronInst.Entry(id); !entry.Next.IsZero() {
		config.DB.Model(&model.Task{}).Where("id = ?", task.ID).Update("next_run", entry.Next)
	}
	return id
}

// AddTask registers an enabled task with the scheduler.
func AddTask(task model.Task) {
	if task.Status != "enabled" || task.CronExpr == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	registerTask(task)
}

// RemoveTask removes a task from the scheduler.
func RemoveTask(taskID uint) {
	mu.Lock()
	defer mu.Unlock()
	if id, ok := entryMap[taskID]; ok {
		cronInst.Remove(id)
		delete(entryMap, taskID)
	}
	config.DB.Model(&model.Task{}).Where("id = ?", taskID).Update("next_run", time.Time{})
}

// ReloadTask re-schedules a task (used after update or toggle).
func ReloadTask(task model.Task) {
	mu.Lock()
	defer mu.Unlock()

	// Remove old entry
	if id, ok := entryMap[task.ID]; ok {
		cronInst.Remove(id)
		delete(entryMap, task.ID)
	}

	// Re-register if enabled
	if task.Status == "enabled" && task.CronExpr != "" {
		registerTask(task)
	} else {
		config.DB.Model(&model.Task{}).Where("id = ?", task.ID).Update("next_run", time.Time{})
	}
}

// runTask executes a task's command and records the result.
func runTask(taskID uint) {
	var task model.Task
	if err := config.DB.First(&task, taskID).Error; err != nil {
		return
	}

	out, err := exec.Command("bash", "-c", task.Command).CombinedOutput()
	result := "success"
	if err != nil {
		result = "failed"
	}

	// Update execution result
	config.DB.Model(&task).Updates(map[string]any{
		"last_run":    time.Now(),
		"last_result": result,
		"last_output": strings.TrimSpace(string(out)),
	})

	// Update next_run time
	mu.Lock()
	if id, ok := entryMap[taskID]; ok {
		if entry := cronInst.Entry(id); !entry.Next.IsZero() {
			config.DB.Model(&task).Update("next_run", entry.Next)
		}
	}
	mu.Unlock()
}
