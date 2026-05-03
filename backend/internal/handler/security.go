package handler

import (
	"opsmanage/internal/config"
	"opsmanage/internal/middleware"
	"opsmanage/internal/model"

	"github.com/gin-gonic/gin"
)

// ListSecurityRules 获取安全规则列表
// @Summary 获取安全规则列表（分页）
// @Tags 安全管理
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /security/rules [get]
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

// CreateSecurityRule 创建安全规则
// @Summary 创建安全规则
// @Description 支持类型: ip_whitelist, ip_blacklist, url_blacklist, ua_blacklist
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateSecurityRuleReq true "规则信息"
// @Success 200 {object} map[string]interface{}
// @Router /security/rules [post]
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
	middleware.ReloadRules()
	addLog("info", "security", "创建安全规则: "+rule.Name+" ("+rule.Type+")")
	success(c, rule)
}

// GetSecurityRule 获取安全规则详情
// @Summary 获取安全规则详情
// @Tags 安全管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Success 200 {object} map[string]interface{}
// @Router /security/rules/{id} [get]
func GetSecurityRule(c *gin.Context) {
	id := c.Param("id")
	var rule model.SecurityRule
	if err := config.DB.First(&rule, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}
	success(c, rule)
}

// UpdateSecurityRule 更新安全规则
// @Summary 更新安全规则
// @Tags 安全管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Success 200 {object} map[string]interface{}
// @Router /security/rules/{id} [put]
func UpdateSecurityRule(c *gin.Context) {
	id := c.Param("id")
	var rule model.SecurityRule
	if err := config.DB.First(&rule, id).Error; err != nil {
		fail(c, 404, "未找到")
		return
	}

	var req struct {
		Name     *string `json:"name"`
		Content  *string `json:"content"`
		Priority *int    `json:"priority"`
		Remark   *string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	updates := map[string]any{}
	if req.Name != nil { updates["name"] = *req.Name }
	if req.Content != nil { updates["content"] = *req.Content }
	if req.Priority != nil { updates["priority"] = *req.Priority }
	if req.Remark != nil { updates["remark"] = *req.Remark }

	if len(updates) > 0 {
		config.DB.Model(&rule).Updates(updates)
	}
	config.DB.First(&rule, id)
	middleware.ReloadRules()
	success(c, rule)
}

// DeleteSecurityRule 删除安全规则
// @Summary 删除安全规则
// @Tags 安全管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Success 200 {object} map[string]interface{}
// @Router /security/rules/{id} [delete]
func DeleteSecurityRule(c *gin.Context) {
	id := c.Param("id")
	if err := config.DB.Delete(&model.SecurityRule{}, id).Error; err != nil {
		fail(c, 500, "删除失败")
		return
	}
	middleware.ReloadRules()
	success(c, nil)
}

// ToggleSecurityRule 切换规则状态
// @Summary 启用/禁用安全规则
// @Tags 安全管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "规则ID"
// @Success 200 {object} map[string]interface{}
// @Router /security/rules/{id}/toggle [post]
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
	middleware.ReloadRules()
	success(c, rule)
}
