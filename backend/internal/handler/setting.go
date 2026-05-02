package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"

	"github.com/gin-gonic/gin"
)

func GetSettings(c *gin.Context) {
	var settings []model.Setting
	config.DB.Find(&settings)
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	success(c, result)
}

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

func GetSettingByKey(c *gin.Context) {
	key := c.Param("key")
	var setting model.Setting
	if err := config.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		fail(c, 404, "设置项不存在")
		return
	}
	success(c, setting)
}
