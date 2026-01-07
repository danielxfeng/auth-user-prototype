package routers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
)

func TestDevRouterResetDebugMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	util.InitLogger(slog.LevelError)

	prevCfg := config.Cfg
	config.Cfg = &config.Config{GinMode: "debug"}
	t.Cleanup(func() {
		config.Cfg = prevCfg
	})

	db.ConnectDB("file::memory:?cache=shared")
	t.Cleanup(func() {
		if db.DB != nil {
			sqlDB, err := db.DB.DB()
			if err == nil {
				sqlDB.Close()
			}
			db.DB = nil
		}
	})

	if err := db.DB.Create(&db.User{Username: "tester", Email: "tester@example.com"}).Error; err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	router := gin.New()
	DevRouter(router.Group("/api/dev"))

	req := httptest.NewRequest(http.MethodGet, "/api/dev/reset", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload["message"] != "ok" {
		t.Fatalf("unexpected response message: %v", payload)
	}

	var count int64
	if err := db.DB.Model(&db.User{}).Count(&count).Error; err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected user table to be empty, found %d rows", count)
	}
}

func TestDevRouterNoopOutsideDebug(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevCfg := config.Cfg
	config.Cfg = &config.Config{GinMode: "release"}
	t.Cleanup(func() {
		config.Cfg = prevCfg
	})

	router := gin.New()
	DevRouter(router.Group("/api/dev"))

	req := httptest.NewRequest(http.MethodGet, "/api/dev/reset", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 when debug routes disabled, got %d", resp.Code)
	}
}
