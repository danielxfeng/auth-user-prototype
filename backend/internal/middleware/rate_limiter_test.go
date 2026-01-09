package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
)

func TestAllowRequestResetsAfterWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := middleware.NewRateLimiter(30*time.Millisecond, 2)
	clientID := "client-1"

	if !rl.AllowRequest(clientID) {
		t.Fatalf("expected first request to pass")
	}
	if !rl.AllowRequest(clientID) {
		t.Fatalf("expected second request to pass within window")
	}
	if rl.AllowRequest(clientID) {
		t.Fatalf("expected third request to be blocked within window")
	}

	time.Sleep(40 * time.Millisecond)

	if !rl.AllowRequest(clientID) {
		t.Fatalf("expected requests to be allowed after window resets")
	}
}

func TestRateLimitMiddlewareBlocksAfterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := middleware.NewRateLimiter(50*time.Millisecond, 1)

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(rl.RateLimit())
	r.GET("/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req1 := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req1.RemoteAddr = "198.51.100.10:1234"
	resp1 := httptest.NewRecorder()
	r.ServeHTTP(resp1, req1)
	if resp1.Code != http.StatusOK {
		t.Fatalf("expected first request status 200, got %d", resp1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/limited", nil)
	req2.RemoteAddr = "198.51.100.10:5678"
	resp2 := httptest.NewRecorder()
	r.ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d body=%s", resp2.Code, resp2.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(resp2.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] != "Too many requests" {
		t.Fatalf("unexpected error payload: %v", body)
	}
}

func TestRateLimitMiddlewareSkipsOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := middleware.NewRateLimiter(50*time.Millisecond, 1)

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(rl.RateLimit())
	r.OPTIONS("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.GET("/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	reqOptions := httptest.NewRequest(http.MethodOptions, "/limited", nil)
	reqOptions.RemoteAddr = "198.51.100.20:9999"
	for i := 0; i < 3; i++ {
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, reqOptions)
		if resp.Code != http.StatusNoContent {
			t.Fatalf("expected OPTIONS to bypass limiter with 204, got %d", resp.Code)
		}
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/limited", nil)
	reqGet.RemoteAddr = "198.51.100.20:9999"
	respGet1 := httptest.NewRecorder()
	r.ServeHTTP(respGet1, reqGet)
	if respGet1.Code != http.StatusOK {
		t.Fatalf("expected first GET 200 after OPTIONS calls, got %d", respGet1.Code)
	}

	respGet2 := httptest.NewRecorder()
	r.ServeHTTP(respGet2, reqGet)
	if respGet2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second GET to be rate limited with 429, got %d", respGet2.Code)
	}
}

func TestRateLimitMiddlewareUsesClientSpecificCounters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl := middleware.NewRateLimiter(100*time.Millisecond, 1)

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(rl.RateLimit())
	r.GET("/limited", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	reqClientA := httptest.NewRequest(http.MethodGet, "/limited", nil)
	reqClientA.RemoteAddr = "203.0.113.1:5000"
	respA1 := httptest.NewRecorder()
	r.ServeHTTP(respA1, reqClientA)
	if respA1.Code != http.StatusOK {
		t.Fatalf("expected client A first request 200, got %d", respA1.Code)
	}

	reqClientB := httptest.NewRequest(http.MethodGet, "/limited", nil)
	reqClientB.RemoteAddr = "203.0.113.2:5000"
	respB := httptest.NewRecorder()
	r.ServeHTTP(respB, reqClientB)
	if respB.Code != http.StatusOK {
		t.Fatalf("expected client B request 200, got %d body=%s", respB.Code, respB.Body.String())
	}

	reqClientA2 := httptest.NewRequest(http.MethodGet, "/limited", nil)
	reqClientA2.RemoteAddr = "203.0.113.1:6000"
	respA2 := httptest.NewRecorder()
	r.ServeHTTP(respA2, reqClientA2)
	if respA2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected client A second request 429, got %d", respA2.Code)
	}
}
