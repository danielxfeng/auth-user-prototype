package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

var testDep = testutil.NewTestDependency(nil, nil, nil, nil)

type testAuthService struct {
	returnCode int
}

func (ts *testAuthService) GetDependency() *dependency.Dependency {
	return testDep
}

func (ts *testAuthService) ValidateUserToken(ctx context.Context, tokenString string, userID uint) error {
	switch ts.returnCode {
	case 200:
		return nil
	case 401:
		return authError.NewAuthError(401, "invalid token")
	default:
		return fmt.Errorf("unexpected error")
	}
}

func newTestAuthService(returnCode int) middleware.AuthService {
	return &testAuthService{returnCode: returnCode}
}

const notSet = "notSet"

func TestAuth(t *testing.T) {
	validToken, err := jwt.SignUserToken(testDep, 1)
	if err != nil {
		t.Fatalf("failed to sign test token, err: %v", err)
	}

	testCases := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{name: "valid token", token: validToken, expectedStatus: 200},
		{name: "invalid token", token: "aaa", expectedStatus: 401},
		{name: "token not set", token: notSet, expectedStatus: 401},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := testutil.NewMiddlewareTestRouter(middleware.Auth(newTestAuthService(tc.expectedStatus)), nil)
			req, _ := http.NewRequest("POST", "/middleware-test", nil)

			if tc.token != notSet {
				req.Header.Add("Authorization", fmt.Sprintf("%s%s", middleware.PrefixBearer, tc.token))
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Fatalf("expected: %d, got: %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
