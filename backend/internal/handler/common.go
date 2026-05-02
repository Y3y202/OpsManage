package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "" || r.Header.Get("Origin") == "http://"+r.Host || r.Header.Get("Origin") == "https://"+r.Host
	},
}

func success(c *gin.Context, data any) {
	c.JSON(200, gin.H{"code": 200, "msg": "操作成功", "data": data})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(200, gin.H{"code": code, "msg": msg})
}

func pageResult(c *gin.Context, list any, total int64) {
	c.JSON(200, gin.H{"code": 200, "msg": "ok", "data": gin.H{"list": list, "total": total}})
}

func paginate(c *gin.Context, query *gorm.DB) *gorm.DB {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	return query.Offset(offset).Limit(pageSize)
}
