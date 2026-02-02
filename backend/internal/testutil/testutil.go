package testutil

import (
	"io"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func NewTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

func NewTestConfig() *config.Config {
	return &config.Config{
		GinMode:                         "test",
		DbAddress:                       "inmemory://test",
		JwtSecret:                       "test-jwt-secret",
		UserTokenExpiry:                 5,
		OauthStateTokenExpiry:           5,
		GoogleClientId:                  "test-google-client-id",
		GoogleClientSecret:              "test-google-client-secret",
		GoogleRedirectUri:               "test-google-redirect-uri",
		FrontendUrl:                     "http://localhost:5173",
		TwoFaUrlPrefix:                  "otpauth://totp/Transcendence?secret=",
		TwoFaTokenExpiry:                5,
		RedisURL:                        "",
		IsRedisEnabled:                  false,
		UserTokenAbsoluteExpiry:         2592000,
		Port:                            3003,
		RateLimiterDurationInSec:        5,
		RateLimiterRequestLimit:         10,
		RateLimiterCleanupIntervalInSec: 10,
	}
}

func NewTestDependency(cfg *config.Config, db *gorm.DB, redis *redis.Client, logger *slog.Logger) *dependency.Dependency {
	if cfg == nil {
		cfg = NewTestConfig()
	}

	if logger == nil {
		logger = NewTestLogger()
	}

	if redis != nil {
		cfg.IsRedisEnabled = true
		if cfg.RedisURL == "" {
			cfg.RedisURL = "redis://test"
		}
	}
	return dependency.NewDependency(cfg, db, redis, logger)
}

func NewMiddlewareTestRouter(middleware1 gin.HandlerFunc, middleware2 gin.HandlerFunc) *gin.Engine {
	r := gin.New()

	if middleware1 != nil {
		r.Use(middleware1)
	}

	if middleware2 != nil {
		r.Use(middleware2)
	}

	r.POST("/middleware-test", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})

	return r
}

func NewIntegrationTestRouter(dep *dependency.Dependency, handlers ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()

	for _, handler := range handlers {
		r.Use(handler)
	}

	return r
}
