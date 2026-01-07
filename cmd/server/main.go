package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/routers"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"

	"log/slog"

	sloggin "github.com/samber/slog-gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
)

func SetupRouter(logger *slog.Logger) *gin.Engine {
	r := gin.New()

	logConfig := sloggin.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
	}

	r.Use(sloggin.NewWithConfig(logger, logConfig))
	r.Use(middleware.PanicHandler())
	r.Use(middleware.ErrorHandler())

	return r
}

func main() {
	// config
	godotenv.Load()
	config.LoadConfig()

	// logger
	util.InitLogger(slog.LevelInfo)

	// router
	r := SetupRouter(util.Logger)
	routers.UsersRouter(r.Group("/api/user"))
	routers.DevRouter(r.Group("/api/dev"))

	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.Run(":3003")
}
