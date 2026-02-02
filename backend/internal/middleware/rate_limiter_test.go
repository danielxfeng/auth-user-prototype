package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
)

func TestRateLimiter(t *testing.T) {
	testCases := []struct {
		name           string
		duration       time.Duration
		limit          int
		sleep          time.Duration
		methods        []string
		remoteAddrs    []string
		expectedStatus []int
		needOptions    bool
	}{
		{
			name:           "blocks after limit",
			duration:       100 * time.Millisecond,
			limit:          2,
			methods:        []string{http.MethodPost, http.MethodPost, http.MethodPost},
			remoteAddrs:    []string{"203.0.113.1:1000", "203.0.113.1:1000", "203.0.113.1:1000"},
			expectedStatus: []int{200, 200, 429},
		},
		{
			name:           "resets after window",
			duration:       30 * time.Millisecond,
			limit:          1,
			sleep:          60 * time.Millisecond,
			methods:        []string{http.MethodPost, http.MethodPost, http.MethodPost},
			remoteAddrs:    []string{"198.51.100.3:9999", "198.51.100.3:9999", "198.51.100.3:9999"},
			expectedStatus: []int{200, 429, 200},
		},
		{
			name:           "options bypass",
			duration:       100 * time.Millisecond,
			limit:          1,
			methods:        []string{http.MethodOptions, http.MethodOptions, http.MethodOptions, http.MethodPost, http.MethodPost},
			remoteAddrs:    []string{"203.0.113.2:5555", "203.0.113.2:5555", "203.0.113.2:5555", "203.0.113.2:5555", "203.0.113.2:5555"},
			expectedStatus: []int{204, 204, 204, 200, 429},
			needOptions:    true,
		},
		{
			name:           "client isolation",
			duration:       100 * time.Millisecond,
			limit:          1,
			methods:        []string{http.MethodPost, http.MethodPost, http.MethodPost},
			remoteAddrs:    []string{"203.0.113.10:5000", "203.0.113.11:5000", "203.0.113.10:6000"},
			expectedStatus: []int{200, 200, 429},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rl := middleware.NewRateLimiter(tc.duration, tc.limit, time.Minute)
			r := testutil.NewMiddlewareTestRouter(rl.RateLimit(), middleware.ErrorHandler())
			if tc.needOptions {
				r.OPTIONS("/middleware-test", func(c *gin.Context) {
					c.Status(204)
				})
			}

			for i := 0; i < len(tc.methods); i++ {
				req, _ := http.NewRequest(tc.methods[i], "/middleware-test", nil)
				req.RemoteAddr = tc.remoteAddrs[i]

				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if w.Code != tc.expectedStatus[i] {
					t.Fatalf("expected: %d, got: %d", tc.expectedStatus[i], w.Code)
				}

				if tc.sleep > 0 && i == 1 {
					time.Sleep(tc.sleep)
				}
			}
		})
	}
}

func TestAllowRequest(t *testing.T) {
	type step struct {
		sleep  time.Duration
		client string
		expect bool
	}

	testCases := []struct {
		name     string
		duration time.Duration
		limit    int
		cleanup  time.Duration
		steps    []step
	}{
		{
			name:     "limit reached",
			duration: 50 * time.Millisecond,
			limit:    1,
			cleanup:  time.Minute,
			steps: []step{
				{client: "client-a", expect: true},
				{client: "client-a", expect: false},
			},
		},
		{
			name:     "cleanup path",
			duration: 50 * time.Millisecond,
			limit:    1,
			cleanup:  1 * time.Millisecond,
			steps: []step{
				{client: "client-a", expect: true},
				{sleep: 2 * time.Millisecond},
				{client: "client-b", expect: true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rl := middleware.NewRateLimiter(tc.duration, tc.limit, tc.cleanup)

			for _, s := range tc.steps {
				if s.sleep > 0 {
					time.Sleep(s.sleep)
					continue
				}

				allowed := rl.AllowRequest(s.client)
				if allowed != s.expect {
					t.Fatalf("expected: %v, got: %v", s.expect, allowed)
				}
			}
		})
	}
}
