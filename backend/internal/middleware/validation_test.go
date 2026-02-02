package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
)

type testPayload struct {
	Name string `json:"name" validate:"required"`
}

func TestValidateBody(t *testing.T) {
	dto.InitValidator()

	testCases := []struct {
		name           string
		payload        string
		expectedStatus int
	}{
		{name: "success", payload: `{"name":"ok"}`, expectedStatus: 200},
		{name: "validation error", payload: `{}`, expectedStatus: 400},
		{name: "invalid json", payload: `{`, expectedStatus: 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := testutil.NewMiddlewareTestRouter(
				middleware.ValidateBody[testPayload](),
				middleware.ErrorHandler(),
			)
			reqBody := strings.NewReader(tc.payload)
			req, _ := http.NewRequest(http.MethodPost, "/middleware-test", reqBody)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected: %d, got: %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
