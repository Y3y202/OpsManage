package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"

	"github.com/gin-gonic/gin"
)

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

func GetDatabase(c *gin.Context) {
	id := c.Param("id")
	var db model.Database
	if err := config.DB.First(&db, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, db)
}

func UpdateDatabase(c *gin.Context) {
	id := c.Param("id")
	var db model.Database
	if err := config.DB.First(&db, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	delete(updates, "id")
	delete(updates, "created_at")
	config.DB.Model(&db).Updates(updates)
	config.DB.First(&db, id)
	success(c, db)
}

func DeleteDatabase(c *gin.Context) {
	id := c.Param("id")
	if err := config.DB.Delete(&model.Database{}, id).Error; err != nil {
		fail(c, 500, "删除失败")
		return
	}
	success(c, nil)
}

func StartDatabase(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Database{}).Where("id = ?", id).Update("status", "running")
	success(c, gin.H{"status": "running"})
}

func StopDatabase(c *gin.Context) {
	id := c.Param("id")
	config.DB.Model(&model.Database{}).Where("id = ?", id).Update("status", "stopped")
	success(c, gin.H{"status": "stopped"})
}
