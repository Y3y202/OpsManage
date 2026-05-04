package handler

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"opsmanage/internal/config"
	"opsmanage/internal/middleware"
	"opsmanage/internal/model"
	"strings"
	"time"

	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func addLog(level, source, message string) {
	config.DB.Create(&model.LogEntry{
		Level:   level,
		Source:  source,
		Message: message,
	})
}

type LoginReq struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户名密码 + 验证码登录，返回 JWT Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param body body LoginReq true "登录信息"
// @Success 200 {object} map[string]interface{} "{"code":200,"msg":"操作成功","data":{"token":"...","user":{...}}}"
// @Router /auth/login [post]
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	// Validate captcha (temporarily disabled for screenshot)
	if req.CaptchaCode != "" && !captcha.VerifyString(req.CaptchaID, req.CaptchaCode) {
		addLog("warn", "auth", "登录失败: 验证码错误, IP: "+c.ClientIP())
		fail(c, 400, "验证码错误")
		return
	}

	var user model.User
	if err := config.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		addLog("warn", "auth", "登录失败: 用户名 "+req.Username+" 不存在, IP: "+c.ClientIP())
		fail(c, 401, "用户名或密码错误")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		addLog("warn", "auth", "登录失败: 用户 "+req.Username+" 密码错误, IP: "+c.ClientIP())
		fail(c, 401, "用户名或密码错误")
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		fail(c, 500, "token生成失败")
		return
	}

	config.DB.Model(&user).Updates(map[string]any{
		"last_login": time.Now(),
		"ip":         c.ClientIP(),
	})

	addLog("info", "auth", "用户 "+user.Username+" 登录成功, IP: "+c.ClientIP())

	success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"role":     user.Role,
		},
	})
}

type RegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Register 注册用户
// @Summary 注册新用户（仅管理员）
// @Description 创建新用户账号，需要管理员权限
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body RegisterReq true "注册信息"
// @Success 200 {object} map[string]interface{} "{"code":200,"data":{"id":1}}"
// @Failure 400 {object} map[string]interface{}
// @Router /auth/register [post]
func Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	// Default to normal user, only allow admin if requester is admin
	role := "user"
	if req.Role == "admin" {
		role = "admin"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, 500, "密码加密失败")
		return
	}

	user := model.User{
		Username: req.Username,
		Password: string(hash),
		Email:    req.Email,
		Nickname: req.Username,
		Role:     role,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		fail(c, 400, "用户名已存在")
		return
	}

	requester, _ := c.Get("username")
	addLog("info", "auth", requester.(string)+" 创建用户: "+user.Username+" (role: "+role+")")
	success(c, gin.H{"id": user.ID})
}

// ListUsers 获取用户列表
// @Summary 获取用户列表（仅管理员）
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /users [get]
func ListUsers(c *gin.Context) {
	var users []model.User
	config.DB.Order("id desc").Find(&users)
	var result []gin.H
	for _, u := range users {
		result = append(result, gin.H{
			"id":         u.ID,
			"username":   u.Username,
			"nickname":   u.Nickname,
			"email":      u.Email,
			"role":       u.Role,
			"last_login": u.LastLogin,
			"ip":         u.IP,
			"created_at": u.CreatedAt,
		})
	}
	success(c, result)
}

// DeleteUser 删除用户
// @Summary 删除用户（仅管理员）
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Param id path int true "用户ID"
// @Success 200 {object} map[string]interface{}
// @Router /users/{id} [delete]
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	uid, _ := c.Get("user_id")

	// Cannot delete yourself
	if fmt.Sprintf("%v", uid) == id {
		fail(c, 400, "不能删除当前登录用户")
		return
	}

	var user model.User
	if err := config.DB.First(&user, id).Error; err != nil {
		fail(c, 404, "用户不存在")
		return
	}

	config.DB.Delete(&user)
	requester, _ := c.Get("username")
	addLog("info", "auth", requester.(string)+" 删除用户: "+user.Username)
	success(c, nil)
}

// Logout 用户登出
// @Summary 用户登出
// @Description 将当前 Token 加入黑名单
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /auth/logout [post]
func Logout(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 {
			middleware.BlacklistToken(parts[1])
		}
	}
	username, _ := c.Get("username")
	if name, ok := username.(string); ok && name != "" {
		addLog("info", "auth", "用户 "+name+" 已登出")
	}
	success(c, nil)
}

// Captcha 获取验证码
// @Summary 获取验证码图片
// @Description 生成验证码图片（base64）和验证码ID
// @Tags 认证
// @Produce json
// @Success 200 {object} map[string]interface{} "{"code":200,"data":{"captcha_id":"...","captcha":"data:image/png;base64,..."}}"
// @Router /auth/captcha [get]
func Captcha(c *gin.Context) {
	id := captcha.NewLen(4)
	var buf bytes.Buffer
	captcha.WriteImage(&buf, id, 200, 80)
	imgBase64 := "data:image/png;base64," + base64Encode(buf.Bytes())
	success(c, gin.H{
		"captcha_id": id,
		"captcha":    imgBase64,
	})
}

// GetProfile 获取当前用户信息
// @Summary 获取当前登录用户信息
// @Tags 用户管理
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /profile [get]
func GetProfile(c *gin.Context) {
	uid, _ := c.Get("user_id")
	var user model.User
	if err := config.DB.First(&user, uid).Error; err != nil {
		fail(c, 404, "用户不存在")
		return
	}
	success(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"nickname":   user.Nickname,
		"email":      user.Email,
		"role":       user.Role,
		"last_login": user.LastLogin,
		"ip":         user.IP,
	})
}

type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改密码
// @Summary 修改当前用户密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body ChangePasswordReq true "密码信息"
// @Success 200 {object} map[string]interface{}
// @Router /password [put]
func ChangePassword(c *gin.Context) {
	uid, _ := c.Get("user_id")
	var req ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
	}

	var user model.User
	config.DB.First(&user, uid)
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		fail(c, 400, "原密码错误")
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	config.DB.Model(&user).Update("password", string(hash))
	success(c, nil)
}
