package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
)

const rateLimiterPath = "/middleware-test"

func newRateLimiterRouter(rl *middleware.RateLimiter) *gin.Engine {
	r := testutil.NewIntegrationTestRouter(nil, middleware.ErrorHandler(), rl.RateLimit())

	r.POST(rateLimiterPath, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.OPTIONS(rateLimiterPath, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	return r
}

func doRequest(r http.Handler, method string, ip string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, rateLimiterPath, nil)
	req.RemoteAddr = ip
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func assertErrorMessage(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["error"] != expected {
		t.Fatalf("unexpected error payload: %v", body)
	}
}

func TestRateLimiterBlocksAfterLimit(t *testing.T) {
	rl := middleware.NewRateLimiter(100*time.Millisecond, 2, time.Minute)
	r := newRateLimiterRouter(rl)

	resp1 := doRequest(r, http.MethodPost, "203.0.113.1:1000")
	if resp1.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp1.Code)
	}

	resp2 := doRequest(r, http.MethodPost, "203.0.113.1:1000")
	if resp2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp2.Code)
	}

	blocked := doRequest(r, http.MethodPost, "203.0.113.1:1000")
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, blocked.Code)
	}
	assertErrorMessage(t, blocked, "Too many requests")
}

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	rl := middleware.NewRateLimiter(30*time.Millisecond, 1, time.Minute)
	r := newRateLimiterRouter(rl)

	resp1 := doRequest(r, http.MethodPost, "198.51.100.3:9999")
	if resp1.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp1.Code)
	}

	resp2 := doRequest(r, http.MethodPost, "198.51.100.3:9999")
	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, resp2.Code)
	}

	time.Sleep(60 * time.Millisecond)

	resp3 := doRequest(r, http.MethodPost, "198.51.100.3:9999")
	if resp3.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp3.Code)
	}
}

func TestRateLimiterOptionsBypass(t *testing.T) {
	rl := middleware.NewRateLimiter(100*time.Millisecond, 1, time.Minute)
	r := newRateLimiterRouter(rl)

	for i := 0; i < 3; i++ {
		resp := doRequest(r, http.MethodOptions, "203.0.113.2:5555")
		if resp.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, resp.Code)
		}
	}

	resp1 := doRequest(r, http.MethodPost, "203.0.113.2:5555")
	if resp1.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp1.Code)
	}

	resp2 := doRequest(r, http.MethodPost, "203.0.113.2:5555")
	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, resp2.Code)
	}
}

func TestRateLimiterClientIsolation(t *testing.T) {
	rl := middleware.NewRateLimiter(100*time.Millisecond, 1, time.Minute)
	r := newRateLimiterRouter(rl)

	resp1 := doRequest(r, http.MethodPost, "203.0.113.10:5000")
	if resp1.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp1.Code)
	}

	resp2 := doRequest(r, http.MethodPost, "203.0.113.11:5000")
	if resp2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp2.Code)
	}

	resp3 := doRequest(r, http.MethodPost, "203.0.113.10:6000")
	if resp3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, resp3.Code)
	}
}
