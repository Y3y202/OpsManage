package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"

	"github.com/gin-gonic/gin"
)

// ListDatabases 获取数据库列表
// @Summary 获取数据库实例列表（分页）
// @Tags 数据库管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /databases [get]
func ListDatabases(c *gin.Context) {
	var dbs []model.Database
	var total int64
	query := config.DB.Model(&model.Database{})
	query.Count(&total)
	paginate(c, query).Order("id desc").Find(&dbs)
	pageResult(c, dbs, total)
}

type CreateDatabaseReq struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Version  string `json:"version"`
}

// CreateDatabase 创建数据库实例
// @Summary 创建数据库实例
// @Description 支持 MySQL、PostgreSQL、Redis 三种类型
// @Tags 数据库管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateDatabaseReq true "数据库信息"
// @Success 200 {object} map[string]interface{}
// @Router /databases [post]
func CreateDatabase(c *gin.Context) {
	var req CreateDatabaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	if req.Host == "" {
		req.Host = "127.0.0.1"
	}

	switch req.Type {
	case "mysql":
		if req.Port == 0 {
			req.Port = 3306
		}
	case "postgresql":
		if req.Port == 0 {
			req.Port = 5432
		}
	case "redis":
		if req.Port == 0 {
			req.Port = 6379
		}
	default:
		fail(c, 400, "不支持的数据库类型，仅支持 mysql, postgresql, redis")
		return
	}

	db := model.Database{
		Name:     req.Name,
		Type:     req.Type,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		Status:   "running",
		Version:  req.Version,
	}

	if err := config.DB.Create(&db).Error; err != nil {
		fail(c, 500, "创建失败: "+err.Error())
		return
	}
	addLog("info", "database", "创建数据库实例: "+db.Name+" ("+db.Type+")")
	success(c, db)
}

// GetDatabase 获取数据库详情
// @Summary 获取数据库实例详情
// @Tags 数据库管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "数据库ID"
// @Success 200 {object} map[string]interface{}
// @Router /databases/{id} [get]
func GetDatabase(c *gin.Context) {
	id := c.Param("id")
	var db model.Database
	if err := config.DB.First(&db, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, db)
}

// UpdateDatabase 更新数据库实例
// @Summary 更新数据库实例配置
// @Tags 数据库管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "数据库ID"
// @Success 200 {object} map[string]interface{}
// @Router /databases/{id} [put]
func UpdateDatabase(c *gin.Context) {
	id := c.Param("id")
	var db model.Database
	if err := config.DB.First(&db, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var req struct {
		Name     *string `json:"name"`
		Host     *string `json:"host"`
		Port     *int    `json:"port"`
		Username *string `json:"username"`
		Password *string `json:"password"`
		Version  *string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	updates := map[string]any{}
	if req.Name != nil { updates["name"] = *req.Name }
	if req.Host != nil { updates["host"] = *req.Host }
	if req.Port != nil { updates["port"] = *req.Port }
	if req.Username != nil { updates["username"] = *req.Username }
	if req.Password != nil { updates["password"] = *req.Password }
	if req.Version != nil { updates["version"] = *req.Version }

	if len(updates) > 0 {
		config.DB.Model(&db).Updates(updates)
	}
	config.DB.First(&db, id)
	success(c, db)
}

// DeleteDatabase 删除数据库实例
// @Summary 删除数据库实例
// @Tags 数据库管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "数据库ID"
// @Success 200 {object} map[string]interface{}
// @Router /databases/{id} [delete]
func DeleteDatabase(c *gin.Context) {
	id := c.Param("id")
	if err := config.DB.Delete(&model.Database{}, id).Error; err != nil {
		fail(c, 500, "删除失败")
		return
	}
	success(c, nil)
}

// StartDatabase 启动数据库
// @Summary 启动数据库实例
// @Tags 数据库管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "数据库ID"
// @Success 200 {object} map[string]interface{}
// @Router /databases/{id}/start [post]
func StartDatabase(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Database{}).Where("id = ?", id).Update("status", "running")
	success(c, gin.H{"status": "running"})
}

// StopDatabase 停止数据库
// @Summary 停止数据库实例
// @Tags 数据库管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "数据库ID"
// @Success 200 {object} map[string]interface{}
// @Router /databases/{id}/stop [post]
func StopDatabase(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Database{}).Where("id = ?", id).Update("status", "stopped")
	success(c, gin.H{"status": "stopped"})
}
