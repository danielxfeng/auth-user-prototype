package main

import (
	"net/http"
	"os"
	"time"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/paularynty/transcendence/auth-service-go/docs"
	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/routers"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"

	"log/slog"

	sloggin "github.com/samber/slog-gin"

	"github.com/gin-contrib/cors"

	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
)

func SetupRouter(logger *slog.Logger) *gin.Engine {
	r := gin.New()

	logConfig := sloggin.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"http://localhost:4173",
			"https://localhost:5173",
			"https://c2r5p11.hive.fi:5173",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	rateLimiter := middleware.NewRateLimiter(60*time.Second, 1000)
	r.Use(rateLimiter.RateLimit())

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
	_ = godotenv.Load()

	config.LoadConfig()

	// logger
	util.InitLogger(slog.LevelInfo)

	// validator
	dto.InitValidator()

	// database
	db.ConnectDB(config.Cfg.DbAddress)
	defer db.CloseDB()

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

	if err := r.Run(":3003"); err != nil {
		util.Logger.Error("failed to start server", "err", err)
		os.Exit(1)
	}
}
