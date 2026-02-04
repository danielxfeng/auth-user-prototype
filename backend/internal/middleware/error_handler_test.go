package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
)

type testStruct struct {
	UserId int `json:"userID" validate:"required"`
}

func errorGenerator(code int) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch code {
		case 200:
			c.Next()
		case 401:
			_ = c.AbortWithError(401, authError.NewAuthError(401, "invalid user"))
		case 400:
			v := validator.New()
			s := testStruct{}

			err := v.Struct(s)
			if err == nil {
				panic("expected validation error, got nil")
			}

			_ = c.AbortWithError(400, err)
		case 500:
			_ = c.AbortWithError(500, fmt.Errorf("unknown error"))
		default:
			panic("panic test")
		}
	}
}

func TestErrorHandler(t *testing.T) {
	testCases := []struct {
		name         string
		code         int
		expectedCode int
	}{
		{name: "normal case", code: 200, expectedCode: 200},
		{name: "400 error", code: 400, expectedCode: 400},
		{name: "401 error", code: 401, expectedCode: 401},
		{name: "500 error", code: 500, expectedCode: 500},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := testutil.NewMiddlewareTestRouter(errorGenerator(tc.code), middleware.ErrorHandler())
			w := httptest.NewRecorder()

			req, _ := http.NewRequest("POST", "/middleware-test", nil)
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Fatalf("expected: %d, got: %d", tc.expectedCode, w.Code)
			}
		})
	}
}
