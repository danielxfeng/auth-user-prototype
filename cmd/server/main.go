package main

import (
	"net/http"

   swaggerfiles "github.com/swaggo/files"
   ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/routers"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
	_ "github.com/paularynty/transcendence/auth-service-go/docs"

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

	r.Use(middleware.PanicHandler())
	r.Use(sloggin.NewWithConfig(logger, logConfig))
	r.Use(middleware.ErrorHandler())

	return r
}

// @title Auth Service API
// @version 1.0
// @description Auth service for Transcendence
// @BasePath /api
func main() {
	// config
	godotenv.Load()
	config.LoadConfig()

	// logger
	util.InitLogger(slog.LevelInfo)

	// validator
	dto.InitValidator()

	// router
	r := SetupRouter(util.Logger)
	routers.UsersRouter(r.Group("/api/users"))
	routers.DevRouter(r.Group("/api/dev"))

	// Health check
	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Swagger
	r.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	r.Run(":3003")
}
