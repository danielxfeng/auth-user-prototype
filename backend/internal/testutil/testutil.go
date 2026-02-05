package testutil

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/service"
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
		DbAddress:                       "file::memory:?cache=shared",
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
		userID := c.GetUint("userID")
		token := c.GetString("token")

		c.JSON(200, gin.H{
			"userID": userID,
			"token":  token,
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

func GetSafeTestDBName(dbName string, testName string) string {
	if !strings.Contains(dbName, "file::memory") {
		return dbName 
	}

	safeName := strings.NewReplacer("/", "_", " ", "_").Replace(testName)
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", safeName)
}

func NewTestUserService(t *testing.T) (*service.UserService, *gorm.DB) {
	t.Helper()

	cfg := NewTestConfig()
	logger := NewTestLogger()

	cfg.DbAddress = GetSafeTestDBName(cfg.DbAddress, t.Name())

	myDB, err := db.GetDB(cfg.DbAddress, logger)
	if err != nil {
		t.Fatalf("failed to init test db, err: %v", err)
	}
	t.Cleanup(func() {
		db.CloseDB(myDB, logger)
	})
	db.ResetDB(myDB, logger)

	dep := NewTestDependency(cfg, myDB, nil, logger)

	userService, err := service.NewUserService(dep)
	if err != nil {
		t.Fatalf("failed to create user service, err: %v", err)
	}

	return userService, myDB
}

func CreateUser(t *testing.T, myDB *gorm.DB, username, email string, avatar *string) db.User {
	t.Helper()

	user := db.User{
		Username: username,
		Email:    email,
		Avatar:   avatar,
	}
	if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
		t.Fatalf("failed to create user, err: %v", err)
	}
	return user
}
