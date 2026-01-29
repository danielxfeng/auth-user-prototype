package jwt_test

import (
	"errors"
	"testing"
	"time"

	libjwt "github.com/golang-jwt/jwt/v5"

	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

func setupTokenDep(t *testing.T) *dependency.Dependency {
	t.Helper()
	cfg := testutil.NewTestConfig()
	cfg.JwtSecret = "test-secret-key"
	cfg.UserTokenExpiry = 3600
	cfg.OauthStateTokenExpiry = 120
	cfg.TwoFaTokenExpiry = 300
	return testutil.NewTestDependency(cfg, nil, nil, nil)
}

func TestTokenRoundTrip(t *testing.T) {
	dep := setupTokenDep(t)

	cases := []struct {
		name          string
		sign          func() (string, error)
		validate      func(string) (any, error)
		assert        func(t *testing.T, claims any)
		expectedError error
	}{
		{
			name: "UserToken",
			sign: func() (string, error) {
				return jwt.SignUserToken(dep, 42)
			},
			validate: func(token string) (any, error) {
				return jwt.ValidateUserTokenGeneric(dep, token)
			},
			assert: func(t *testing.T, claims any) {
				parsed := claims.(*dto.UserJwtPayload)
				if parsed.UserID != 42 {
					t.Fatalf("expected user id 42, got %d", parsed.UserID)
				}
				if parsed.Type != jwt.UserTokenType {
					t.Fatalf("expected claim type %q, got %q", jwt.UserTokenType, parsed.Type)
				}
				if parsed.ExpiresAt == nil || parsed.ExpiresAt.Before(time.Now()) {
					t.Fatalf("expected future expiration, got %v", parsed.ExpiresAt)
				}
			},
		},
		{
			name: "OauthStateToken",
			sign: func() (string, error) {
				return jwt.SignOauthStateToken(dep)
			},
			validate: func(token string) (any, error) {
				return jwt.ValidateOauthStateToken(dep, token)
			},
			assert: func(t *testing.T, claims any) {
				parsed := claims.(*dto.OauthStateJwtPayload)
				if parsed.Type != jwt.GoogleOAuthStateType {
					t.Fatalf("expected oauth state type %q, got %q", jwt.GoogleOAuthStateType, parsed.Type)
				}
			},
		},
		{
			name: "TwoFASetupToken",
			sign: func() (string, error) {
				return jwt.SignTwoFASetupToken(dep, 7, "secret")
			},
			validate: func(token string) (any, error) {
				return jwt.ValidateTwoFASetupToken(dep, token)
			},
			assert: func(t *testing.T, claims any) {
				parsed := claims.(*dto.TwoFaSetupJwtPayload)
				if parsed.Secret != "secret" {
					t.Fatalf("expected secret to be propagated, got %q", parsed.Secret)
				}
			},
		},
		{
			name: "UserTokenRejectsWrongType",
			sign: func() (string, error) {
				return jwt.SignTwoFAToken(dep, 10)
			},
			validate: func(token string) (any, error) {
				return jwt.ValidateUserTokenGeneric(dep, token)
			},
			expectedError: libjwt.ErrTokenInvalidClaims,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := tc.sign()
			if err != nil {
				t.Fatalf("sign returned error: %v", err)
			}

			claims, err := tc.validate(token)
			if tc.expectedError != nil {
				if !errors.Is(err, tc.expectedError) {
					t.Fatalf("expected %v, got %v", tc.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validate returned error: %v", err)
			}

			if tc.assert != nil {
				tc.assert(t, claims)
			}
		})
	}
}
