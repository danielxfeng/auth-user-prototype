package service

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
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

func setupConfig() {
	config.Cfg = &config.Config{
		JwtSecret:             "test-secret",
		UserTokenExpiry:       3600,
		OauthStateTokenExpiry: 600,
		GoogleClientId:        "test-client-id",
		GoogleClientSecret:    "test-client-secret",
		GoogleRedirectUri:     "http://localhost:8080/callback",
		FrontendUrl:           "http://localhost:3000",
		TwoFaUrlPrefix:        "otpauth://totp/Transcendence?secret=",
		TwoFaTokenExpiry:      600,
	}

	// Mock logger to discard output during tests
	util.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors
	}))
}

func TestMain(m *testing.M) {
	setupConfig()
	code := m.Run()
	os.Exit(code)
}
