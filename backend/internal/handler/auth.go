package handler

import (
	"crypto/rand"
	"encoding/base64"
	"opsmanage/internal/config"
	"opsmanage/internal/middleware"
	"opsmanage/internal/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func addLog(level, source, message string) {
	config.DB.Create(&model.LogEntry{
		Level:   level,
		Source:  source,
		Message: message,
	})
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
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

	config.DB.Model(&user).Updates(map[string]interface{}{
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
}

func Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "参数错误")
		return
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
		Role:     "admin",
	}

	if err := config.DB.Create(&user).Error; err != nil {
		fail(c, 400, "用户名已存在")
		return
	}

	success(c, gin.H{"id": user.ID})
}

func Logout(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 {
			middleware.BlacklistToken(parts[1])
		}
	}
	username, _ := c.Get("username")
	addLog("info", "auth", "用户 "+username.(string)+" 已登出")
	success(c, nil)
}

func Captcha(c *gin.Context) {
	b := make([]byte, 32)
	rand.Read(b)
	id := base64.URLEncoding.EncodeToString(b)[:16]
	c.JSON(200, gin.H{
		"captcha_id": id,
		"captcha":    strings.Repeat("●", 4),
	})
}

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
