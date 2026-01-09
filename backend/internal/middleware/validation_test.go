package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
)

func TestValidateBodyPassesValidPayloadAndStoresInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dto.InitValidator()

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.ValidateBody[dto.UserName]())
	r.POST("/ok", func(c *gin.Context) {
		val, exists := c.Get("validatedBody")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "validatedBody missing"})
			return
		}

		name, ok := val.(dto.UserName)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "wrong type"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"username": name.Username})
	})

	payload := dto.UserName{Username: "valid_user"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/ok", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var respBody map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respBody["username"] != "valid_user" {
		t.Fatalf("expected username to propagate from validatedBody, got %v", respBody)
	}
}

func TestValidateBodyHandlesBindErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dto.InitValidator()

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.ValidateBody[dto.UserName]())
	r.POST("/bad", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/bad", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	var respBody map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respBody["error"] == nil {
		t.Fatalf("expected error field in response, got %v", respBody)
	}
}

func TestValidateBodyReturnsValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dto.InitValidator()

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.ValidateBody[dto.UserName]())
	r.POST("/fail", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// too short username triggers validator rule
	req := httptest.NewRequest(http.MethodPost, "/fail", bytes.NewBufferString(`{"username":"ab"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	var respBody map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	errorsField, ok := respBody["error"].([]any)
	if !ok || len(errorsField) == 0 {
		t.Fatalf("expected validation errors array, got %v", respBody)
	}
}
