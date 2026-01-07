package util_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
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

	token, err := util.SignUserToken(42)
	if err != nil {
		t.Fatalf("SignUserToken returned error: %v", err)
	}

	claims, err := util.ValidateUserTokenGeneric(token)
	if err != nil {
		t.Fatalf("ValidateUserTokenGeneric returned error: %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("expected user id 42, got %d", claims.UserID)
	}

	if claims.Type != util.UserTokenType {
		t.Fatalf("expected claim type %q, got %q", util.UserTokenType, claims.Type)
	}

	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		t.Fatalf("expected future expiration, got %v", claims.ExpiresAt)
	}
}

func TestValidateUserTokenRejectsWrongType(t *testing.T) {
	cleanup := setupTokenConfig(t)
	defer cleanup()

	token, err := util.SignTwoFAToken(10)
	if err != nil {
		t.Fatalf("SignTwoFAToken returned error: %v", err)
	}

	_, err = util.ValidateUserTokenGeneric(token)
	if !errors.Is(err, jwt.ErrTokenInvalidClaims) {
		t.Fatalf("expected ErrTokenInvalidClaims, got %v", err)
	}
}

func TestValidateOauthStateToken(t *testing.T) {
	cleanup := setupTokenConfig(t)
	defer cleanup()

	token, err := util.SignOauthStateToken()
	if err != nil {
		t.Fatalf("SignOauthStateToken returned error: %v", err)
	}

	claims, err := util.ValidateOauthStateToken(token)
	if err != nil {
		t.Fatalf("ValidateOauthStateToken returned error: %v", err)
	}

	if claims.Type != util.GoogleOAuthStateType {
		t.Fatalf("expected oauth state type %q, got %q", util.GoogleOAuthStateType, claims.Type)
	}
}

func TestValidateTwoFASetupToken(t *testing.T) {
	cleanup := setupTokenConfig(t)
	defer cleanup()

	token, err := util.SignTwoFASetupToken(7, "secret")
	if err != nil {
		t.Fatalf("SignTwoFASetupToken returned error: %v", err)
	}

	claims, err := util.ValidateTwoFASetupToken(token)
	if err != nil {
		t.Fatalf("ValidateTwoFASetupToken returned error: %v", err)
	}

	if claims.Secret != "secret" {
		t.Fatalf("expected secret to be propagated, got %q", claims.Secret)
	}
}
