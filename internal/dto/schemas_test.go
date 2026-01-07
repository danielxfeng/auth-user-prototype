package dto_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
)

func init() {
	dto.InitValidator()
}

func TestUserAvatarMustBeURL(t *testing.T) {
	avatar := "avatar.png"
	payload := dto.User{
		UserName: dto.UserName{Username: "valid_user"},
		Email:    "user@example.com",
		Avatar:   &avatar,
	}

	if err := dto.Validate.Struct(&payload); err == nil {
		t.Fatalf("expected non-URL avatar to be rejected by url validator")
	}

	validAvatar := "https://example.com/avatar.png"
	payload.Avatar = &validAvatar

	if err := dto.Validate.Struct(&payload); err != nil {
		t.Fatalf("expected URL avatar to pass validation, got error: %v", err)
	}
}

func TestTwoFAChallengeRequiresNumericOTP(t *testing.T) {
	invalid := dto.TwoFAChallengeRequest{TwoFACode: "AB12CD", SessionToken: "session-token"}
	if err := dto.Validate.Struct(&invalid); err == nil {
		t.Fatalf("expected alphanumeric code to fail numeric validator")
	}

	valid := dto.TwoFAChallengeRequest{TwoFACode: "123456", SessionToken: "session-token"}
	if err := dto.Validate.Struct(&valid); err != nil {
		t.Fatalf("expected numeric code to pass validation, got error: %v", err)
	}
}

func TestTwoFAPendingUserResponseRequiresTaggedFields(t *testing.T) {
	payload := dto.TwoFAPendingUserResponse{
		Message:      "ANY_VALUE",
		TwoFASecret:  "secret",
		SessionToken: "session-token",
		TwoFAUrl:     "custom://value",
	}

	if err := dto.Validate.Struct(&payload); err != nil {
		t.Fatalf("expected arbitrary message/twoFaUrl to be accepted, got error: %v", err)
	}

	payload.TwoFASecret = ""
	if err := dto.Validate.Struct(&payload); err == nil {
		t.Fatalf("expected required fields to be enforced when empty")
	}
}

func TestUserJWTTypeAllowsAnyString(t *testing.T) {
	payload := dto.UserJwtPayload{
		UserID: 1,
		Type:   "OTHER",
	}

	if err := dto.Validate.Struct(&payload); err != nil {
		t.Fatalf("expected arbitrary type value to be accepted, got error: %v", err)
	}
}

func TestUsersResponseMarshalsAsObjectWithSlice(t *testing.T) {
	payload := dto.UsersResponse{Users: []dto.SimpleUser{}}

	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal users response: %v", err)
	}

	expected := "{\"users\":[]}"
	if string(bytes) != expected {
		t.Fatalf("expected users response to marshal to %s, got %s", expected, string(bytes))
	}
}

func TestTrimValidationStripsWhitespace(t *testing.T) {
	type payload struct {
		Value string `validate:"required,trim,min=3"`
	}

	data := &payload{Value: "  foo  "}
	if err := dto.Validate.Struct(data); err != nil {
		t.Fatalf("expected trimmed value to pass validation, got error: %v", err)
	}

	if data.Value != "foo" {
		t.Fatalf("expected trim validator to remove outer spaces, got %q", data.Value)
	}

	tooShort := &payload{Value: "  a "}
	if err := dto.Validate.Struct(tooShort); err == nil {
		t.Fatalf("expected trimmed value shorter than min to fail validation")
	}
}

func TestUsernameValidatorRules(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid", "valid_user", false},
		{"ValidTrimmed", "  valid-user  ", false},
		{"TooShort", "ab", true},
		{"TooShortAfterTrim", "  ab  ", true},
		{"ContainsSpace", "user name", true},
		{"IllegalChars", "user@name", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := &dto.UserName{Username: tc.input}
			err := dto.Validate.Struct(payload)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected username %q to be invalid", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected username %q to be valid, got error: %v", tc.input, err)
			}

			if payload.Username != strings.TrimSpace(tc.input) {
				t.Fatalf("expected username to be trimmed to %q, got %q", strings.TrimSpace(tc.input), payload.Username)
			}
		})
	}
}

func TestPasswordValidatorRules(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"ValidBasic", "Abc123", false},
		{"ValidSymbols", "pass,#$%", false},
		{"TooShort", "ab", true},
		{"TooShortAfterTrim", "  ab  ", true},
		{"ContainsSpace", "pass word", true},
		{"DisallowedChar", "bad~pass", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := &dto.Password{Password: tc.input}
			err := dto.Validate.Struct(payload)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected password %q to be invalid", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected password %q to be valid, got error: %v", tc.input, err)
			}

			if payload.Password != strings.TrimSpace(tc.input) {
				t.Fatalf("expected password to be trimmed to %q, got %q", strings.TrimSpace(tc.input), payload.Password)
			}
		})
	}
}

func TestIdentifierValidatorAcceptsUsernameOrEmail(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Username", "valid_user", false},
		{"Email", "user@example.com", false},
		{"TrimmedEmail", "  user@example.com  ", false},
		{"Invalid", "???", true},
		{"TooShort", "ab", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := &dto.Identifier{Identifier: tc.input}
			err := dto.Validate.Struct(payload)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected identifier %q to be invalid", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected identifier %q to be valid, got error: %v", tc.input, err)
			}

			if payload.Identifier != strings.TrimSpace(tc.input) {
				t.Fatalf("expected identifier to be trimmed to %q, got %q", strings.TrimSpace(tc.input), payload.Identifier)
			}
		})
	}
}

func TestCreateUserRequestValidation(t *testing.T) {
	avatar := "https://example.com/avatar.png"
	valid := &dto.CreateUserRequest{
		User: dto.User{
			UserName: dto.UserName{Username: "valid_user"},
			Email:    "user@example.com",
			Avatar:   &avatar,
		},
		Password: dto.Password{Password: "Valid123"},
	}

	if err := dto.Validate.Struct(valid); err != nil {
		t.Fatalf("expected create user request to be valid, got error: %v", err)
	}

	invalid := &dto.CreateUserRequest{
		User: dto.User{
			UserName: dto.UserName{Username: "valid_user"},
			Email:    "user@example.com",
			Avatar:   &avatar,
		},
		Password: dto.Password{Password: "no~"},
	}

	if err := dto.Validate.Struct(invalid); err == nil {
		t.Fatalf("expected create user request with disallowed password to fail validation")
	}
}

func TestLoginUserRequestValidation(t *testing.T) {
	valid := &dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "valid_user"},
		Password:   dto.Password{Password: "Valid123"},
	}

	if err := dto.Validate.Struct(valid); err != nil {
		t.Fatalf("expected login request to be valid, got error: %v", err)
	}

	invalid := &dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "??"},
		Password:   dto.Password{Password: "Valid123"},
	}

	if err := dto.Validate.Struct(invalid); err == nil {
		t.Fatalf("expected login request with invalid identifier to fail validation")
	}
}

func TestTwoFAConfirmRequestValidation(t *testing.T) {
	valid := &dto.TwoFAConfirmRequest{TwoFACode: "123456", SetupToken: "token"}
	if err := dto.Validate.Struct(valid); err != nil {
		t.Fatalf("expected valid 2FA confirm request to pass, got error: %v", err)
	}

	invalid := &dto.TwoFAConfirmRequest{TwoFACode: "ABC123", SetupToken: "token"}
	if err := dto.Validate.Struct(invalid); err == nil {
		t.Fatalf("expected non-numeric 2FA code to fail validation")
	}
}
