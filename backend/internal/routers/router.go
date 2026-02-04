package routers

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	sloggin "github.com/samber/slog-gin"
)

func SetupRouter(dep *dependency.Dependency) *gin.Engine {
	r := gin.New()

	r.Use(middleware.PanicHandler())

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

	rateLimiter := middleware.NewRateLimiter(time.Duration(dep.Cfg.RateLimiterDurationInSec)*time.Second, dep.Cfg.RateLimiterRequestLimit, time.Duration(dep.Cfg.RateLimiterCleanupIntervalInSec)*time.Second)
	r.Use(rateLimiter.RateLimit())

	r.Use(sloggin.NewWithConfig(dep.Logger, logConfig))
	r.Use(middleware.ErrorHandler())

	return r
}
