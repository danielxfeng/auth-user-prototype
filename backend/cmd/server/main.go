package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/paularynty/transcendence/auth-service-go/docs"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/routers"
	"github.com/paularynty/transcendence/auth-service-go/internal/service"
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

// @title Auth Service API
// @version 1.0
// @description Auth service
// @BasePath /api
func main() {
	// config
	_ = godotenv.Load()

	// logger
	logger := util.GetLogger(slog.LevelDebug)

	// init dependency
	dep, err := dependency.InitDependency(logger)
	if err != nil {
		util.LogFatalErr(logger, err, "failed to create dependency")
	}
	defer dependency.CloseDependency(dep)

	// validator
	dto.InitValidator()

	// create services
	userService, err := service.NewUserService(dep)
	if err != nil {
		util.LogFatalErr(logger, err, "failed to create user service")
	}

	// router
	r := SetupRouter(dep)
	routers.UsersRouter(r.Group("/api/users"), userService)

	// Health check
	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Swagger
	r.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", dep.Cfg.Port),
		Handler: r,
	}

	// Start server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			util.LogFatalErr(logger, err, "failed to start server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // consume the signal, blocking here
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		util.LogFatalErr(logger, err, "server forced to shutdown")
	}
	logger.Info("server exiting")
}
