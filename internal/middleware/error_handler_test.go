package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
)

func TestErrorHandlerReturnsAuthErrorPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/auth", func(c *gin.Context) {
		c.AbortWithError(http.StatusUnauthorized, middleware.NewAuthError(http.StatusUnauthorized, "Invalid or expired token"))
	})

	req := httptest.NewRequest(http.MethodGet, "/auth", nil)
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
		t.Fatalf("unexpected error payload: %v", body)
	}
}

func TestErrorHandlerDifferentErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/unknown", func(c *gin.Context) {
		c.AbortWithError(http.StatusTeapot, errors.New("boom"))
	})

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusTeapot {
		t.Fatalf("expected status 418, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["error"] != "Internal Server Error" {
		t.Fatalf("unexpected error payload: %v", body)
	}
}

func TestPanicHandlerReturnsJSON500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.PanicHandler())
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["error"] != "Internal Server Error" {
		t.Fatalf("unexpected error payload: %v", body)
	}
}
