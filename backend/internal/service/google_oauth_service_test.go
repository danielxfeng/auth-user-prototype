package service_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"cloud.google.com/go/auth/credentials/idtoken"
	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
	"github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/service"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
	"gorm.io/gorm"
)

func TestHandleGoogleOAuthCallback(t *testing.T) {
	t.Run("invalid state token", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		redirect := userService.HandleGoogleOAuthCallback(context.Background(), "code", "bad-state")
		u, err := url.Parse(redirect)
		if err != nil {
			t.Fatalf("failed to parse redirect url, err: %v", err)
		}
		if u.Query().Get("error") == "" {
			t.Fatalf("expected error query param")
		}
		if u.Query().Get("token") != "" {
			t.Fatalf("did not expect token query param")
		}
	})

	t.Run("exchange code failure", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		origExchange := service.ExchangeCodeForTokens
		t.Cleanup(func() { service.ExchangeCodeForTokens = origExchange })
		service.ExchangeCodeForTokens = func(dep *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
			return nil, errors.New("exchange failed")
		}

		state, err := jwt.SignOauthStateToken(userService.Dep)
		if err != nil {
			t.Fatalf("failed to sign state token, err: %v", err)
		}

		redirect := userService.HandleGoogleOAuthCallback(context.Background(), "code", state)
		u, err := url.Parse(redirect)
		if err != nil {
			t.Fatalf("failed to parse redirect url, err: %v", err)
		}
		if u.Query().Get("error") == "" {
			t.Fatalf("expected error query param")
		}
	})

	t.Run("fetch google user info failure", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		origExchange := service.ExchangeCodeForTokens
		origFetch := service.FetchGoogleUserInfo
		t.Cleanup(func() {
			service.ExchangeCodeForTokens = origExchange
			service.FetchGoogleUserInfo = origFetch
		})

		service.ExchangeCodeForTokens = func(dep *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
			return &idtoken.Payload{Subject: "gid", Claims: map[string]any{}}, nil
		}
		service.FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
			return nil, authError.NewAuthError(400, "bad token")
		}

		state, err := jwt.SignOauthStateToken(userService.Dep)
		if err != nil {
			t.Fatalf("failed to sign state token, err: %v", err)
		}

		redirect := userService.HandleGoogleOAuthCallback(context.Background(), "code", state)
		u, err := url.Parse(redirect)
		if err != nil {
			t.Fatalf("failed to parse redirect url, err: %v", err)
		}
		if u.Query().Get("error") == "" {
			t.Fatalf("expected error query param")
		}
	})

	t.Run("existing google oauth id logs in", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		origExchange := service.ExchangeCodeForTokens
		t.Cleanup(func() { service.ExchangeCodeForTokens = origExchange })
		service.ExchangeCodeForTokens = func(dep *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
			return &idtoken.Payload{
				Subject: "gid-1",
				Claims: map[string]any{
					"email": "gid@example.com",
					"name":  "Gid",
				},
			}, nil
		}

		googleID := "gid-1"
		user := db.User{
			Username:      "giduser",
			Email:         "gid@example.com",
			GoogleOauthID: &googleID,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		state, err := jwt.SignOauthStateToken(userService.Dep)
		if err != nil {
			t.Fatalf("failed to sign state token, err: %v", err)
		}

		redirect := userService.HandleGoogleOAuthCallback(context.Background(), "code", state)
		u, err := url.Parse(redirect)
		if err != nil {
			t.Fatalf("failed to parse redirect url, err: %v", err)
		}
		if u.Query().Get("token") == "" {
			t.Fatalf("expected token query param")
		}
		if u.Query().Get("error") != "" {
			t.Fatalf("did not expect error query param")
		}
	})

	t.Run("existing email fails to link google account", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		origExchange := service.ExchangeCodeForTokens
		t.Cleanup(func() { service.ExchangeCodeForTokens = origExchange })
		service.ExchangeCodeForTokens = func(dep *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
			return &idtoken.Payload{
				Subject: "gid-2",
				Claims: map[string]any{
					"email": "exists@example.com",
					"name":  "Exists",
				},
			}, nil
		}

		user := db.User{
			Username: "exists",
			Email:    "exists@example.com",
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		state, err := jwt.SignOauthStateToken(userService.Dep)
		if err != nil {
			t.Fatalf("failed to sign state token, err: %v", err)
		}

		redirect := userService.HandleGoogleOAuthCallback(context.Background(), "code", state)
		u, err := url.Parse(redirect)
		if err != nil {
			t.Fatalf("failed to parse redirect url, err: %v", err)
		}
		if u.Query().Get("error") == "" {
			t.Fatalf("expected error query param")
		}
	})

	t.Run("new user created", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		origExchange := service.ExchangeCodeForTokens
		t.Cleanup(func() { service.ExchangeCodeForTokens = origExchange })
		service.ExchangeCodeForTokens = func(dep *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
			return &idtoken.Payload{
				Subject: "gid-3",
				Claims: map[string]any{
					"email":   "new@example.com",
					"name":    "New",
					"picture": "https://example.com/a.png",
				},
			}, nil
		}

		state, err := jwt.SignOauthStateToken(userService.Dep)
		if err != nil {
			t.Fatalf("failed to sign state token, err: %v", err)
		}

		redirect := userService.HandleGoogleOAuthCallback(context.Background(), "code", state)
		u, err := url.Parse(redirect)
		if err != nil {
			t.Fatalf("failed to parse redirect url, err: %v", err)
		}
		if u.Query().Get("token") == "" {
			t.Fatalf("expected token query param")
		}
		if u.Query().Get("error") != "" {
			t.Fatalf("did not expect error query param")
		}

		modelUser, err := gorm.G[db.User](myDB).Where("email = ?", "new@example.com").First(context.Background())
		if err != nil {
			t.Fatalf("expected user to be created, err: %v", err)
		}
		if modelUser.GoogleOauthID == nil || *modelUser.GoogleOauthID != "gid-3" {
			t.Fatalf("expected google oauth id to be set")
		}
	})
}
