package service

import (
	"os"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
)

func setupTestDB(testName string) *gorm.DB {
	// Sanitize test name for use as DB identifier
	// Add busy_timeout to reduce locking errors
	// Add _foreign_keys=on to enforce FK constraints
	dbName := "file:" + strings.ReplaceAll(testName, "/", "_") + "?mode=memory&cache=shared&_busy_timeout=5000&_foreign_keys=on"

	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		panic("failed to connect database")
	}

	// Explicitly enable foreign keys for SQLite just in case the DSN parameter isn't enough for the driver version
	db.Exec("PRAGMA foreign_keys = ON")

	err = db.AutoMigrate(&model.User{}, &model.Friend{}, &model.Token{}, &model.HeartBeat{})
	if err != nil {
		panic("failed to migrate database")
	}

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}

	return db
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func newTestDependency(db *gorm.DB, redis *redis.Client, cfgMutators ...func(*config.Config)) *dependency.Dependency {
	cfg := testutil.NewTestConfig()
	for _, mutate := range cfgMutators {
		mutate(cfg)
	}
	logger := testutil.NewTestLogger()
	if redis != nil {
		cfg.IsRedisEnabled = true
		if cfg.RedisURL == "" {
			cfg.RedisURL = "redis://test"
		}
	}
	return dependency.NewDependency(cfg, db, redis, logger)
}

func newTestDependencyWithConfig(cfg *config.Config, db *gorm.DB, redis *redis.Client) *dependency.Dependency {
	if cfg == nil {
		cfg = testutil.NewTestConfig()
	}
	logger := testutil.NewTestLogger()
	if redis != nil {
		cfg.IsRedisEnabled = true
		if cfg.RedisURL == "" {
			cfg.RedisURL = "redis://test"
		}
	}
	return dependency.NewDependency(cfg, db, redis, logger)
}

func setupTestRedis(t *testing.T, cfg *config.Config) (*miniredis.Miniredis, *redis.Client, func()) {
	t.Helper()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	if cfg != nil {
		cfg.RedisURL = "redis://" + mr.Addr()
		cfg.IsRedisEnabled = true
	}

	cleanup := func() {
		_ = client.Close()
		mr.Close()
	}

	return mr, client, cleanup
}
