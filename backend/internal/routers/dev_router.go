package routers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/db"
)

func DevRouter(r *gin.RouterGroup) {
	if config.Cfg.GinMode != "debug" {
		return
	}

	r.GET("/reset", func(c *gin.Context) {
		db.ResetDB()
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
}
