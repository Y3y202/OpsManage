package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"

	"github.com/gin-gonic/gin"
)

// GetSettings 获取所有设置
// @Summary 获取所有系统设置
// @Tags 系统设置
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /settings [get]
func GetSettings(c *gin.Context) {
	var settings []model.Setting
	config.DB.Find(&settings)
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	success(c, result)
}

// UpdateSettings 批量更新设置
// @Summary 批量更新系统设置
// @Tags 系统设置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body map[string]string true "设置键值对"
// @Success 200 {object} map[string]interface{}
// @Router /settings [put]
func UpdateSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	for key, value := range req {
		var setting model.Setting
		if err := config.DB.Where("key = ?", key).First(&setting).Error; err != nil {
			config.DB.Create(&model.Setting{Key: key, Value: value})
		} else {
			config.DB.Model(&setting).Update("value", value)
		}
	}
	success(c, nil)
}

// GetSettingByKey 获取单个设置
// @Summary 根据 Key 获取设置项
// @Tags 系统设置
// @Produce json
// @Security BearerAuth
// @Param key path string true "设置键名"
// @Success 200 {object} map[string]interface{}
// @Router /settings/{key} [get]
func GetSettingByKey(c *gin.Context) {
	key := c.Param("key")
	var setting model.Setting
	if err := config.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		fail(c, 404, "设置项不存在")
		return
	}
	success(c, setting)
}
