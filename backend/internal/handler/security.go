package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/model"

	"github.com/gin-gonic/gin"
)

func ListSecurityRules(c *gin.Context) {
	var rules []model.SecurityRule
	var total int64
	query := config.DB.Model(&model.SecurityRule{})
	query.Count(&total)
	paginate(c, query).Order("priority desc, id desc").Find(&rules)
	pageResult(c, rules, total)
}

type CreateSecurityRuleReq struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Priority int    `json:"priority"`
	Remark   string `json:"remark"`
}

func CreateSecurityRule(c *gin.Context) {
	var req CreateSecurityRuleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	rule := model.SecurityRule{
		Name:     req.Name,
		Type:     req.Type,
		Content:  req.Content,
		Status:   "enabled",
		Priority: req.Priority,
		Remark:   req.Remark,
	}

	if err := config.DB.Create(&rule).Error; err != nil {
		fail(c, 500, "创建失败")
		return
	}
	addLog("info", "security", "创建安全规则: "+rule.Name+" ("+rule.Type+")")
	success(c, rule)
}

func GetSecurityRule(c *gin.Context) {
	id := c.Param("id")
	var rule model.SecurityRule
	if err := config.DB.First(&rule, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, rule)
}

func UpdateSecurityRule(c *gin.Context) {
	id := c.Param("id")
	var rule model.SecurityRule
	if err := config.DB.First(&rule, id).Error; err != nil {
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
	config.DB.Model(&rule).Updates(updates)
	config.DB.First(&rule, id)
	success(c, rule)
}

func DeleteSecurityRule(c *gin.Context) {
	id := c.Param("id")
	if err := config.DB.Delete(&model.SecurityRule{}, id).Error; err != nil {
		fail(c, 500, "删除失败")
		return
	}
	success(c, nil)
}

func ToggleSecurityRule(c *gin.Context) {
	id := c.Param("id")
	var rule model.SecurityRule
	if err := config.DB.First(&rule, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	if rule.Status == "enabled" {
		config.DB.Model(&rule).Update("status", "disabled")
	} else {
		config.DB.Model(&rule).Update("status", "enabled")
	}
	config.DB.First(&rule, id)
	success(c, rule)
}
