package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

func setupAuthDep(t *testing.T) *dependency.Dependency {
	t.Helper()
	cfg := testutil.NewTestConfig()
	cfg.JwtSecret = "test-secret-key"
	cfg.UserTokenExpiry = 3600
	cfg.OauthStateTokenExpiry = 120
	cfg.TwoFaTokenExpiry = 300
	return testutil.NewTestDependency(cfg, nil, nil, nil)
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dep := setupAuthDep(t)

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.Auth(dep))
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
	dep := setupAuthDep(t)

	token, err := jwt.SignUserToken(dep, 99)
	if err != nil {
		t.Fatalf("failed to sign user token: %v", err)
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.Auth(dep))
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
	dep := setupAuthDep(t)

	token, err := jwt.SignTwoFAToken(dep, 10)
	if err != nil {
		t.Fatalf("failed to sign 2fa token: %v", err)
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.Auth(dep))
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
