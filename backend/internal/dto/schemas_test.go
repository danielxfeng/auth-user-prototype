package dto_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
)

func TestUsername_HappyPath(t *testing.T) {
	dto.InitValidator()

	testCases := []struct {
		value         string
		expectedValue string
	}{
		{value: "aaa", expectedValue: "aaa"},
		{value: " aaa  ", expectedValue: "aaa"},
		{value: "aA0_-", expectedValue: "aA0_-"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("schema username happy path test: %q", tc.value), func(t *testing.T) {
			req := &dto.UserName{
				Username: tc.value,
			}

			err := dto.Validate.Struct(req)
			if err != nil {
				t.Fatalf("expected %q, got err: %v", tc.expectedValue, err)
			}

			if req.Username != tc.expectedValue {
				t.Fatalf("expected %q, got %q", tc.expectedValue, req.Username)
			}
		})
	}
}

func TestUsername_Errors(t *testing.T) {
	dto.InitValidator()

	testCases := []string{
		"",                      // empty
		"aa",                    // too short
		" aa",                   // too short after trimming
		" aa ",                  // too short after trimming
		"aa ",                   // too short after trimming
		"a a",                   // invalid char
		"a%a",                   // invalid char
		strings.Repeat("a", 51), // too long
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("schema username test: %q", tc), func(t *testing.T) {
			req := &dto.UserName{
				Username: tc,
			}

			err := dto.Validate.Struct(req)
			if err == nil {
				t.Fatalf("expected error, got : %q", req.Username)
			}

			var ve validator.ValidationErrors
			if !errors.As(err, &ve) {
				t.Fatalf("expected validation error, got %v", err)
			}

			for _, fe := range ve {
				if fe.Field() != "Username" {
					t.Fatalf("expected validation error on Username, got %v", err)
				}
			}
		})
	}
}

type passwordReqFactory func(string) (any, func() string)

func runPasswordTests(t *testing.T, label string, fieldName string, build passwordReqFactory) {
	t.Helper()
	dto.InitValidator()

	validCases := []struct {
		value    string
		expected string
	}{
		{value: "pass123", expected: "pass123"},
		{value: " pass123  ", expected: "pass123"},
		{value: "aA0,.#$%@^;|_!*&?", expected: "aA0,.#$%@^;|_!*&?"},
		{value: strings.Repeat("a", 20), expected: strings.Repeat("a", 20)},
	}

	for _, tc := range validCases {
		t.Run(fmt.Sprintf("%s valid %q", label, tc.value), func(t *testing.T) {
			req, getValue := build(tc.value)

			err := dto.Validate.Struct(req)
			if err != nil {
				t.Fatalf("expected %q, got err: %v", tc.expected, err)
			}

			if got := getValue(); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}

	invalidCases := []string{
		"",                      // empty
		"aaaaa",                 // too short
		" aaaaa ",               // too short after trimming
		"aaa aa",                // invalid char
		"aa{}aa",                // invalid char
		strings.Repeat("a", 21), // too long
	}

	for _, tc := range invalidCases {
		t.Run(fmt.Sprintf("%s invalid %q", label, tc), func(t *testing.T) {
			req, _ := build(tc)

			err := dto.Validate.Struct(req)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			var ve validator.ValidationErrors
			if !errors.As(err, &ve) {
				t.Fatalf("expected validation error, got %v", err)
			}

			for _, fe := range ve {
				if fe.Field() != fieldName {
					t.Fatalf("expected validation error on %s, got %v", fieldName, err)
				}
			}
		})
	}
}

func TestPasswordSchemas(t *testing.T) {
	runPasswordTests(t, "Password", "Password", func(value string) (any, func() string) {
		req := &dto.Password{Password: value}
		return req, func() string { return req.Password }
	})

	runPasswordTests(t, "OldPassword", "OldPassword", func(value string) (any, func() string) {
		req := &dto.OldPassword{OldPassword: value}
		return req, func() string { return req.OldPassword }
	})

	runPasswordTests(t, "NewPassword", "NewPassword", func(value string) (any, func() string) {
		req := &dto.NewPassword{NewPassword: value}
		return req, func() string { return req.NewPassword }
	})
}

func TestIdentifier_HappyPath(t *testing.T) {
	dto.InitValidator()

	testCases := []struct {
		value         string
		expectedValue string
	}{
		{value: "user_01", expectedValue: "user_01"},
		{value: " user@example.com  ", expectedValue: "user@example.com"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("schema identifier happy path test: %q", tc.value), func(t *testing.T) {
			req := &dto.Identifier{Identifier: tc.value}

			err := dto.Validate.Struct(req)
			if err != nil {
				t.Fatalf("expected %q, got err: %v", tc.expectedValue, err)
			}

			if req.Identifier != tc.expectedValue {
				t.Fatalf("expected %q, got %q", tc.expectedValue, req.Identifier)
			}
		})
	}
}

func TestIdentifier_Errors(t *testing.T) {
	dto.InitValidator()

	testCases := []string{
		"",         // empty
		"ab",       // too short for username
		"a a",      // invalid char
		"bad@",     // invalid email
		"@bad.com", // invalid email
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("schema identifier error test: %q", tc), func(t *testing.T) {
			req := &dto.Identifier{Identifier: tc}

			err := dto.Validate.Struct(req)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			var ve validator.ValidationErrors
			if !errors.As(err, &ve) {
				t.Fatalf("expected validation error, got %v", err)
			}

			for _, fe := range ve {
				if fe.Field() != "Identifier" {
					t.Fatalf("expected validation error on Identifier, got %v", err)
				}
			}
		})
	}
}

func TestRequestSchemas_HappyPath(t *testing.T) {
	dto.InitValidator()

	testCases := []struct {
		name string
		req  any
	}{
		{
			name: "CreateUserRequest",
			req: &dto.CreateUserRequest{
				User: dto.User{
					UserName: dto.UserName{Username: "user1"},
					Email:    "user1@example.com",
				},
				Password: dto.Password{Password: "pass123"},
			},
		},
		{
			name: "UpdateUserPasswordRequest",
			req: &dto.UpdateUserPasswordRequest{
				OldPassword: dto.OldPassword{OldPassword: "oldpass"},
				NewPassword: dto.NewPassword{NewPassword: "newpass"},
			},
		},
		{
			name: "LoginUserRequest",
			req: &dto.LoginUserRequest{
				Identifier: dto.Identifier{Identifier: "user1"},
				Password:   dto.Password{Password: "pass123"},
			},
		},
		{
			name: "UpdateUserRequest",
			req: &dto.UpdateUserRequest{
				User: dto.User{
					UserName: dto.UserName{Username: "user1"},
					Email:    "user1@example.com",
				},
			},
		},
		{
			name: "UsernameRequest",
			req:  &dto.UsernameRequest{UserName: dto.UserName{Username: "user1"}},
		},
		{
			name: "SetTwoFARequest",
			req:  &dto.SetTwoFARequest{TwoFA: true},
		},
		{
			name: "DisableTwoFARequest",
			req:  &dto.DisableTwoFARequest{Password: dto.Password{Password: "pass123"}},
		},
		{
			name: "TwoFAConfirmRequest",
			req:  &dto.TwoFAConfirmRequest{TwoFACode: "123456", SetupToken: "setup"},
		},
		{
			name: "TwoFAChallengeRequest",
			req:  &dto.TwoFAChallengeRequest{TwoFACode: "123456", SessionToken: "session"},
		},
		{
			name: "AddNewFriendRequest",
			req:  &dto.AddNewFriendRequest{UserID: 1},
		},
		{
			name: "GoogleOauthCallback",
			req:  &dto.GoogleOauthCallback{Code: "code", State: "state"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := dto.Validate.Struct(tc.req); err != nil {
				t.Fatalf("expected valid %s, got err: %v", tc.name, err)
			}
		})
	}
}
