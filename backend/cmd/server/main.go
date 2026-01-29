package main

import (
	"net/http"
	"os"
	"strings"
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

	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
)

func SetupRouter(dep *dependency.Dependency) *gin.Engine {
	r := gin.New()

	logConfig := sloggin.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
	}

	// A rough CORS
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if origin == "http://localhost:5173" ||
				origin == "http://localhost:4173" {
				return true
			}
			if strings.HasSuffix(origin, ".vercel.app") {
				return true
			}
			return false
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
	r.Use(sloggin.NewWithConfig(dep.Logger, logConfig))
	r.Use(middleware.ErrorHandler())

	return r
}

func initDependency() *dependency.Dependency {
	logger := util.GetLogger(slog.LevelInfo)
	cfg := config.LoadConfigFromEnv()
	myDB := db.GetDB(cfg.DbAddress, logger)
	redis := db.GetRedis(cfg.RedisURL, cfg, logger)

	return dependency.NewDependency(cfg, myDB, redis, logger)
}

// @title Auth Service API
// @version 1.0
// @description Auth service
// @BasePath /api
func main() {
	// config
	_ = godotenv.Load()

	// init dependency
	dep := initDependency()
	defer db.CloseDB(dep.DB, dep.Logger)
	defer db.CloseRedis(dep.Redis, dep.Logger)

	// validator
	dto.InitValidator()

	// router
	r := SetupRouter(dep)
	routers.UsersRouter(r.Group("/api/users"), dep)

	// Health check
	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Swagger
	r.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	if err := r.Run(":3003"); err != nil {
		dep.Logger.Error("failed to start server", "err", err)
		os.Exit(1)
	}
}
