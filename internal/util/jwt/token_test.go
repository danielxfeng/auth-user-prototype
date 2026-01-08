package jwt_test

import (
	"errors"
	"testing"
	"time"

	libjwt "github.com/golang-jwt/jwt/v5"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

func setupTokenConfig(t *testing.T) func() {
	t.Helper()
	prev := config.Cfg
	config.Cfg = &config.Config{
		JwtSecret:             "test-secret-key",
		UserTokenExpiry:       3600,
		OauthStateTokenExpiry: 120,
		TwoFaTokenExpiry:      300,
	}

	return func() {
		config.Cfg = prev
	}
}

func TestSignAndValidateUserToken(t *testing.T) {
	cleanup := setupTokenConfig(t)
	defer cleanup()

	token, err := jwt.SignUserToken(42)
	if err != nil {
		t.Fatalf("SignUserToken returned error: %v", err)
	}

	claims, err := jwt.ValidateUserTokenGeneric(token)
	if err != nil {
		t.Fatalf("ValidateUserTokenGeneric returned error: %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("expected user id 42, got %d", claims.UserID)
	}

	if claims.Type != jwt.UserTokenType {
		t.Fatalf("expected claim type %q, got %q", jwt.UserTokenType, claims.Type)
	}

	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		t.Fatalf("expected future expiration, got %v", claims.ExpiresAt)
	}
}

func TestValidateUserTokenRejectsWrongType(t *testing.T) {
	cleanup := setupTokenConfig(t)
	defer cleanup()

	token, err := jwt.SignTwoFAToken(10)
	if err != nil {
		t.Fatalf("SignTwoFAToken returned error: %v", err)
	}

	_, err = jwt.ValidateUserTokenGeneric(token)
	if !errors.Is(err, libjwt.ErrTokenInvalidClaims) {
		t.Fatalf("expected ErrTokenInvalidClaims, got %v", err)
	}
}

func TestValidateOauthStateToken(t *testing.T) {
	cleanup := setupTokenConfig(t)
	defer cleanup()

	token, err := jwt.SignOauthStateToken()
	if err != nil {
		t.Fatalf("SignOauthStateToken returned error: %v", err)
	}

	claims, err := jwt.ValidateOauthStateToken(token)
	if err != nil {
		t.Fatalf("ValidateOauthStateToken returned error: %v", err)
	}

	if claims.Type != jwt.GoogleOAuthStateType {
		t.Fatalf("expected oauth state type %q, got %q", jwt.GoogleOAuthStateType, claims.Type)
	}
}

func TestValidateTwoFASetupToken(t *testing.T) {
	cleanup := setupTokenConfig(t)
	defer cleanup()

	token, err := jwt.SignTwoFASetupToken(7, "secret")
	if err != nil {
		t.Fatalf("SignTwoFASetupToken returned error: %v", err)
	}

	claims, err := jwt.ValidateTwoFASetupToken(token)
	if err != nil {
		t.Fatalf("ValidateTwoFASetupToken returned error: %v", err)
	}

	if claims.Secret != "secret" {
		t.Fatalf("expected secret to be propagated, got %q", claims.Secret)
	}
}
