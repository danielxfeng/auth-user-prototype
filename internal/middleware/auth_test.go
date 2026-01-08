package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

func setupAuthConfig(t *testing.T) func() {
	t.Helper()
	prev := config.Cfg
	config.Cfg = &config.Config{
		JwtSecret:             "test-secret-key",
		UserTokenExpiry:       3600,
		OauthStateTokenExpiry: 120,
		TwoFaTokenExpiry:      300,
	}

	return func() {
		config.Cfg = prev
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cleanup := setupAuthConfig(t)
	defer cleanup()

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.Auth())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["error"] != "Invalid or expired token" {
		t.Fatalf("unexpected error message: %v", body)
	}
}

func TestAuthMiddlewareAllowsValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cleanup := setupAuthConfig(t)
	defer cleanup()

	token, err := jwt.SignUserToken(99)
	if err != nil {
		t.Fatalf("failed to sign user token: %v", err)
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.Auth())
	r.GET("/protected", func(c *gin.Context) {
		userID, ok := c.Get("userID")
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "missing userID"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"userId": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", middleware.PrefixBearer+token)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if val, ok := body["userId"].(float64); !ok || val != 99 {
		t.Fatalf("expected userId 99, got %v", body["userId"])
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cleanup := setupAuthConfig(t)
	defer cleanup()

	token, err := jwt.SignTwoFAToken(10)
	if err != nil {
		t.Fatalf("failed to sign 2fa token: %v", err)
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.Auth())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", middleware.PrefixBearer+token)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}
}
