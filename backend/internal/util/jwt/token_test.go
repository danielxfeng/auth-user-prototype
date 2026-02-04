package jwt_test

import (
	"testing"

	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

var testDep = testutil.NewTestDependency(nil, nil, nil, nil)

func TestUserToken(t *testing.T) {
	token, err := jwt.SignUserToken(testDep, 3)
	if err != nil {
		t.Fatalf("failed to generate token, got an error: %v", err)
	}

	parsed, err := jwt.ValidateUserTokenGeneric(testDep, token)
	if err != nil {
		t.Fatalf("faled to parse user token, got an error: %v", err)
	}

	if parsed.Type != jwt.UserTokenType {
		t.Fatalf("expected token type: %s, got %s", jwt.UserTokenType, parsed.Type)
	}

	if parsed.UserID != 3 {
		t.Fatalf("expected userID: %d, got %d", 3, parsed.UserID)
	}

	_, err = jwt.ValidateUserTokenGeneric(testDep, "aaa")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	invalidToken, err := jwt.SignOauthStateToken(testDep)
	if err != nil {
		t.Fatalf("failed to generate OauthStateToken")
	}

	_, err = jwt.ValidateUserTokenGeneric(testDep, invalidToken)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestOauthStateToken(t *testing.T) {
	token, err := jwt.SignOauthStateToken(testDep)
	if err != nil {
		t.Fatalf("failed to generate token, got an error: %v", err)
	}

	parsed, err := jwt.ValidateOauthStateToken(testDep, token)
	if err != nil {
		t.Fatalf("faled to parse oauth state token, got an error: %v", err)
	}

	if parsed.Type != jwt.GoogleOAuthStateType {
		t.Fatalf("expected token type: %s, got %s", jwt.GoogleOAuthStateType, parsed.Type)
	}

	_, err = jwt.ValidateOauthStateToken(testDep, "aaa")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	invalidToken, err := jwt.SignUserToken(testDep, 3)
	if err != nil {
		t.Fatalf("failed to generate user token")
	}

	_, err = jwt.ValidateOauthStateToken(testDep, invalidToken)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTwoFASetupToken(t *testing.T) {
	token, err := jwt.SignTwoFASetupToken(testDep, 7, "test-secret")
	if err != nil {
		t.Fatalf("failed to generate token, got an error: %v", err)
	}

	parsed, err := jwt.ValidateTwoFASetupToken(testDep, token)
	if err != nil {
		t.Fatalf("faled to parse two-fa setup token, got an error: %v", err)
	}

	if parsed.Type != jwt.TwoFASetupType {
		t.Fatalf("expected token type: %s, got %s", jwt.TwoFASetupType, parsed.Type)
	}

	if parsed.UserID != 7 {
		t.Fatalf("expected userID: %d, got %d", 7, parsed.UserID)
	}

	if parsed.Secret != "test-secret" {
		t.Fatalf("expected secret: %s, got %s", "test-secret", parsed.Secret)
	}

	_, err = jwt.ValidateTwoFASetupToken(testDep, "aaa")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	invalidToken, err := jwt.SignTwoFAToken(testDep, 7)
	if err != nil {
		t.Fatalf("failed to generate two-fa token")
	}

	_, err = jwt.ValidateTwoFASetupToken(testDep, invalidToken)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestTwoFAToken(t *testing.T) {
	token, err := jwt.SignTwoFAToken(testDep, 11)
	if err != nil {
		t.Fatalf("failed to generate token, got an error: %v", err)
	}

	parsed, err := jwt.ValidateTwoFAToken(testDep, token)
	if err != nil {
		t.Fatalf("faled to parse two-fa token, got an error: %v", err)
	}

	if parsed.Type != jwt.TwoFATokenType {
		t.Fatalf("expected token type: %s, got %s", jwt.TwoFATokenType, parsed.Type)
	}

	if parsed.UserID != 11 {
		t.Fatalf("expected userID: %d, got %d", 11, parsed.UserID)
	}

	_, err = jwt.ValidateTwoFAToken(testDep, "aaa")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	invalidToken, err := jwt.SignTwoFASetupToken(testDep, 11, "test-secret")
	if err != nil {
		t.Fatalf("failed to generate two-fa setup token")
	}

	_, err = jwt.ValidateTwoFAToken(testDep, invalidToken)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
