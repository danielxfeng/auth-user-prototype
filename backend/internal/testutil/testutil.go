package testutil

import (
	"io"
	"log/slog"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func NewTestConfig() *config.Config {
	return &config.Config{
		JwtSecret:               "test-secret",
		UserTokenExpiry:         3600,
		UserTokenAbsoluteExpiry: 2592000,
		OauthStateTokenExpiry:   600,
		GoogleClientId:          "test-client-id",
		GoogleClientSecret:      "test-client-secret",
		GoogleRedirectUri:       "http://localhost:8080/callback",
		FrontendUrl:             "http://localhost:3000",
		TwoFaUrlPrefix:          "otpauth://totp/Transcendence?secret=",
		TwoFaTokenExpiry:        600,
		RedisURL:                "",
		IsRedisEnabled:          false,
	}
}

func NewTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
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
