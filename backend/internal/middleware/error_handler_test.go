package middleware_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
)

func TestErrorHandlerReturnsAuthErrorPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.GET("/auth", func(c *gin.Context) {
		_ = c.AbortWithError(http.StatusUnauthorized, authError.NewAuthError(http.StatusUnauthorized, "Invalid or expired token"))
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
		_ = c.AbortWithError(http.StatusTeapot, errors.New("boom"))
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

func TestValidationMiddlewareReturnsValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dto.InitValidator()

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.ValidateBody[dto.UserName]())
	r.POST("/validate", func(c *gin.Context) {
		// Should not reach when validation fails
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewBufferString(`{"username":"  a(   "}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errorsField, ok := body["error"].([]any)
	if !ok || len(errorsField) != 1 {
		t.Fatalf("expected validation errors array, got %v", body)
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
