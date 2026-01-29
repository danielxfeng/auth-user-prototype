package service

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"cloud.google.com/go/auth/credentials/idtoken"
	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

func TestGetGoogleOAuthURL(t *testing.T) {
	db := setupTestDB(t.Name())
	cfg := testutil.NewTestConfig()
	svc := NewUserService(newTestDependencyWithConfig(cfg, db, nil))
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		authURL, err := svc.GetGoogleOAuthURL(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u, err := url.Parse(authURL)
		if err != nil {
			t.Fatalf("failed to parse url: %v", err)
		}

		q := u.Query()
		if q.Get("client_id") != cfg.GoogleClientId {
			t.Errorf("expected client_id %s, got %s", cfg.GoogleClientId, q.Get("client_id"))
		}
		if q.Get("redirect_uri") != cfg.GoogleRedirectUri {
			t.Errorf("expected redirect_uri %s, got %s", cfg.GoogleRedirectUri, q.Get("redirect_uri"))
		}
		if q.Get("state") == "" {
			t.Error("expected state param")
		}
	})
}

func TestHandleGoogleOAuthCallback_InvalidState(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := NewUserService(newTestDependency(db, nil))
	ctx := context.Background()

	// Helper to parse redirect URL
	parseRedirect := func(redirectURL string) (string, string) {
		u, _ := url.Parse(redirectURL)
		q := u.Query()
		return q.Get("token"), q.Get("error")
	}

	t.Run("InvalidState", func(t *testing.T) {
		redirectURL := svc.HandleGoogleOAuthCallback(ctx, "somecode", "invalidstate")
		token, errMsg := parseRedirect(redirectURL)

		if token != "" {
			t.Error("expected no token")
		}
		if errMsg == "" {
			t.Error("expected error message")
		}
	})

	t.Run("ExpiredState", func(t *testing.T) {
		userToken, _ := jwt.SignUserToken(svc.Dep, 1)
		redirectURL := svc.HandleGoogleOAuthCallback(ctx, "somecode", userToken)

		_, errMsg := parseRedirect(redirectURL)
		if errMsg == "" {
			t.Error("expected error message for wrong token type")
		}
	})
}

func TestHandleGoogleOAuthCallback_Success(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := NewUserService(newTestDependency(db, nil))
	ctx := context.Background()

	// Mock dependencies
	origExchange := ExchangeCodeForTokens
	origFetch := FetchGoogleUserInfo
	defer func() {
		ExchangeCodeForTokens = origExchange
		FetchGoogleUserInfo = origFetch
	}()

	ExchangeCodeForTokens = func(_ *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
		return &idtoken.Payload{Subject: "g123"}, nil
	}

	FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
		return &dto.GoogleUserData{
			ID:    "g123",
			Email: "test@google.com",
			Name:  "Google User",
		}, nil
	}

	state, _ := jwt.SignOauthStateToken(svc.Dep)

	t.Run("NewUser", func(t *testing.T) {
		redirectURL := svc.HandleGoogleOAuthCallback(ctx, "validcode", state)

		u, _ := url.Parse(redirectURL)
		q := u.Query()
		if q.Get("token") == "" {
			t.Error("expected token in redirect")
		}
		if q.Get("error") != "" {
			t.Errorf("unexpected error in redirect: %s", q.Get("error"))
		}

		// Verify user created
		var user model.User
		err := db.Where("email = ?", "test@google.com").First(&user).Error
		if err != nil {
			t.Error("expected user to be created")
		}
		if *user.GoogleOauthID != "g123" {
			t.Error("expected google oauth id to be set")
		}
	})

	t.Run("ExistingUser", func(t *testing.T) {
		// User already created in previous run
		redirectURL := svc.HandleGoogleOAuthCallback(ctx, "validcode", state)

		u, _ := url.Parse(redirectURL)
		q := u.Query()
		if q.Get("token") == "" {
			t.Error("expected token in redirect")
		}
	})

	t.Run("ExistingEmailLink", func(t *testing.T) {
		// Create a non-OAuth user with matching email
		_, _ = svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "emailmatch"}, Email: "linkme@google.com"},
			Password: dto.Password{Password: "p"},
		})

		FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
			return &dto.GoogleUserData{
				ID:    "g_link",
				Email: "linkme@google.com",
				Name:  "Link Me",
			}, nil
		}

		redirectURL := svc.HandleGoogleOAuthCallback(ctx, "validcode", state)
		u, _ := url.Parse(redirectURL)
		q := u.Query()
		if q.Get("token") != "" {
			t.Error("expected no token in redirect")
		}
		if q.Get("error") == "" {
			t.Error("expected error in redirect for same-email linking")
		}

		// Verify existing user is NOT linked
		var user model.User
		err := db.Where("email = ?", "linkme@google.com").First(&user).Error
		if err != nil {
			t.Fatal("expected existing user")
		}
		if user.GoogleOauthID != nil {
			t.Error("expected google oauth id to remain unset")
		}
	})

	t.Run("ExistingEmailWith2FA", func(t *testing.T) {
		// Create a user with 2FA enabled
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "email2fa"}, Email: "2fa@google.com"},
			Password: dto.Password{Password: "p"},
		})
		db.Model(&model.User{}).Where("id = ?", u.ID).Update("two_fa_token", "secret")

		FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
			return &dto.GoogleUserData{
				ID:    "g_2fa",
				Email: "2fa@google.com",
				Name:  "Two Fa",
			}, nil
		}

		redirectURL := svc.HandleGoogleOAuthCallback(ctx, "validcode", state)
		u2, _ := url.Parse(redirectURL)
		q := u2.Query()
		if q.Get("token") != "" {
			t.Error("expected no token in redirect")
		}
		if q.Get("error") == "" {
			t.Error("expected error in redirect for 2FA user")
		}
	})
}

func TestHandleGoogleOAuthCallback_Errors(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := NewUserService(newTestDependency(db, nil))
	ctx := context.Background()

	origExchange := ExchangeCodeForTokens
	origFetch := FetchGoogleUserInfo
	defer func() {
		ExchangeCodeForTokens = origExchange
		FetchGoogleUserInfo = origFetch
	}()

	state, _ := jwt.SignOauthStateToken(svc.Dep)

	t.Run("ExchangeError", func(t *testing.T) {
		ExchangeCodeForTokens = func(_ *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
			return nil, errors.New("exchange failed")
		}

		redirectURL := svc.HandleGoogleOAuthCallback(ctx, "code", state)
		u, _ := url.Parse(redirectURL)
		if u.Query().Get("error") == "" {
			t.Error("expected error message")
		}
	})

	t.Run("FetchError", func(t *testing.T) {
		ExchangeCodeForTokens = func(_ *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
			return &idtoken.Payload{}, nil
		}
		FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
			return nil, errors.New("fetch failed")
		}

		redirectURL := svc.HandleGoogleOAuthCallback(ctx, "code", state)
		u, _ := url.Parse(redirectURL)
		if u.Query().Get("error") == "" {
			t.Error("expected error message")
		}
	})
}

func TestLinkGoogleAccountToExistingUser(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := NewUserService(newTestDependency(db, nil))
	ctx := context.Background()

	u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "linkuser"}, Email: "link@e.com"},
		Password: dto.Password{Password: "p"},
	})

	// Fetch model user
	var modelUser model.User
	db.First(&modelUser, u.ID)

	t.Run("BlockedForSafety", func(t *testing.T) {
		picture := "pic.png"
		googleInfo := &dto.GoogleUserData{
			ID:      "g123",
			Email:   "link@e.com",
			Picture: &picture,
		}

		err := svc.linkGoogleAccountToExistingUser(ctx, &modelUser, googleInfo)
		if err == nil {
			t.Fatal("expected linking to be blocked")
		}
		authErr, ok := err.(*middleware.AuthError)
		if !ok {
			t.Fatalf("expected AuthError, got %T: %v", err, err)
		}
		if authErr.Status != 409 {
			t.Fatalf("expected 409 error, got %d: %v", authErr.Status, authErr)
		}
		if authErr.Message != "same email exists" {
			t.Fatalf("expected safety message, got %q", authErr.Message)
		}
		if modelUser.GoogleOauthID != nil {
			t.Error("expected google id to remain unset")
		}
		if modelUser.Avatar != nil {
			t.Error("expected avatar to remain unchanged")
		}
	})

	t.Run("EmailMismatch", func(t *testing.T) {
		googleInfo := &dto.GoogleUserData{
			ID:    "g456",
			Email: "other@e.com",
		}
		err := svc.linkGoogleAccountToExistingUser(ctx, &modelUser, googleInfo)
		authErr, ok := err.(*middleware.AuthError)
		if err == nil || !ok || authErr.Status != 409 {
			t.Errorf("expected 409 AuthError, got %v", err)
		}
	})

	t.Run("AlreadyLinked", func(t *testing.T) {
		// Linking is currently blocked regardless of state.
		googleInfo := &dto.GoogleUserData{
			ID:    "g789",
			Email: "link@e.com",
		}
		err := svc.linkGoogleAccountToExistingUser(ctx, &modelUser, googleInfo)
		authErr, ok := err.(*middleware.AuthError)
		if err == nil || !ok || authErr.Status != 409 {
			t.Errorf("expected 409 AuthError, got %v", err)
		}
	})
}

func TestCreateNewUserFromGoogleInfo(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := NewUserService(newTestDependency(db, nil))
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		googleInfo := &dto.GoogleUserData{
			ID:    "newg1",
			Email: "new@g.com",
			Name:  "New User",
		}

		user, err := svc.createNewUserFromGoogleInfo(ctx, googleInfo, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.Email != "new@g.com" {
			t.Errorf("expected email new@g.com, got %s", user.Email)
		}
		if user.Username != "G_newg1" {
			t.Errorf("expected username G_newg1, got %s", user.Username)
		}
	})

	t.Run("DuplicateUsernameRetry", func(t *testing.T) {
		// Create a user that conflicts with the default google username
		_, _ = svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "G_gdup"}, Email: "existing@e.com"},
			Password: dto.Password{Password: "p"},
		})

		googleInfo := &dto.GoogleUserData{
			ID:    "gdup",
			Email: "unique@g.com",
		}

		user, err := svc.createNewUserFromGoogleInfo(ctx, googleInfo, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should have generated a random UUID based username
		if user.Username == "G_gdup" {
			t.Error("expected random username on collision")
		}
		if user.Email != "unique@g.com" {
			t.Error("expected correct email")
		}
	})
}

func TestHandleGoogleOAuthCallback_DBError(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := NewUserService(newTestDependency(db, nil))
	ctx := context.Background()

	origExchange := ExchangeCodeForTokens
	origFetch := FetchGoogleUserInfo
	defer func() {
		ExchangeCodeForTokens = origExchange
		FetchGoogleUserInfo = origFetch
	}()

	state, _ := jwt.SignOauthStateToken(svc.Dep)

	ExchangeCodeForTokens = func(_ *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
		return &idtoken.Payload{Subject: "g123"}, nil
	}

	FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
		return &dto.GoogleUserData{
			ID:    "g123",
			Email: "test@google.com",
			Name:  "Google User",
		}, nil
	}

	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	redirectURL := svc.HandleGoogleOAuthCallback(ctx, "code", state)
	u, _ := url.Parse(redirectURL)
	if u.Query().Get("error") == "" {
		t.Error("expected error message on closed db")
	}
}

func TestHandleGoogleOAuthCallback_LinkError(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := NewUserService(newTestDependency(db, nil))
	ctx := context.Background()

	origExchange := ExchangeCodeForTokens
	origFetch := FetchGoogleUserInfo
	defer func() {
		ExchangeCodeForTokens = origExchange
		FetchGoogleUserInfo = origFetch
	}()

	state, _ := jwt.SignOauthStateToken(svc.Dep)

	ExchangeCodeForTokens = func(_ *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
		return &idtoken.Payload{Subject: "new_g_id"}, nil
	}

	FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
		return &dto.GoogleUserData{
			ID:    "new_g_id",
			Email: "test@google.com",
			Name:  "Google User",
		}, nil
	}

	// Create user with SAME email but DIFFERENT google ID (already linked)
	googleID := "old_g_id"
	svc.Dep.DB.Create(&model.User{
		Username:      "existing",
		Email:         "test@google.com",
		GoogleOauthID: &googleID,
	})

	redirectURL := svc.HandleGoogleOAuthCallback(ctx, "code", state)
	u, _ := url.Parse(redirectURL)
	if u.Query().Get("error") == "" {
		t.Error("expected error message for link failure")
	}
}
