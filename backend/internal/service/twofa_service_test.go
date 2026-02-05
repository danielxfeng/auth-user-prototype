package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
	"github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestStartTwoFaSetup(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		_, err := userService.StartTwoFaSetup(context.Background(), 9999)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 404 {
			t.Fatalf("expected status 404, got %d", authErr.Status)
		}
	})

	t.Run("already enabled", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		secret := "enabled-secret"
		user := db.User{
			Username:   "user1",
			Email:      "user1@example.com",
			TwoFAToken: &secret,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		_, err := userService.StartTwoFaSetup(context.Background(), user.ID)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("google oauth user", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		googleID := "gid-1"
		user := db.User{
			Username:      "user2",
			Email:         "user2@example.com",
			GoogleOauthID: &googleID,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		_, err := userService.StartTwoFaSetup(context.Background(), user.ID)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{
			Username: "user3",
			Email:    "user3@example.com",
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		resp, err := userService.StartTwoFaSetup(context.Background(), user.ID)
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if resp.TwoFASecret == "" || resp.SetupToken == "" || resp.TwoFaUri == "" {
			t.Fatalf("expected non-empty 2FA setup response")
		}

		modelUser, err := gorm.G[db.User](myDB).Where("id = ?", user.ID).First(context.Background())
		if err != nil {
			t.Fatalf("failed to query user, err: %v", err)
		}
		if modelUser.TwoFAToken == nil || *modelUser.TwoFAToken != "pre-"+resp.TwoFASecret {
			t.Fatalf("expected two_fa_token to be prefixed")
		}
	})
}

func TestConfirmTwoFaSetup(t *testing.T) {
	t.Run("invalid setup token", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		_, err := userService.ConfirmTwoFaSetup(context.Background(), 1, &dto.TwoFAConfirmRequest{
			TwoFACode:  "123456",
			SetupToken: "bad-token",
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("setup token user mismatch", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		token, err := jwt.SignTwoFASetupToken(userService.Dep, 123, "secret")
		if err != nil {
			t.Fatalf("failed to sign setup token, err: %v", err)
		}

		_, err = userService.ConfirmTwoFaSetup(context.Background(), 999, &dto.TwoFAConfirmRequest{
			TwoFACode:  "123456",
			SetupToken: token,
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("setup not initiated", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{
			Username: "user1",
			Email:    "user1@example.com",
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		token, err := jwt.SignTwoFASetupToken(userService.Dep, user.ID, "secret")
		if err != nil {
			t.Fatalf("failed to sign setup token, err: %v", err)
		}

		_, err = userService.ConfirmTwoFaSetup(context.Background(), user.ID, &dto.TwoFAConfirmRequest{
			TwoFACode:  "123456",
			SetupToken: token,
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("invalid 2FA code", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		secret := "JBSWY3DPEHPK3PXP"
		preSecret := "pre-" + secret
		user := db.User{
			Username:   "user2",
			Email:      "user2@example.com",
			TwoFAToken: &preSecret,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		token, err := jwt.SignTwoFASetupToken(userService.Dep, user.ID, secret)
		if err != nil {
			t.Fatalf("failed to sign setup token, err: %v", err)
		}

		_, err = userService.ConfirmTwoFaSetup(context.Background(), user.ID, &dto.TwoFAConfirmRequest{
			TwoFACode:  "000000",
			SetupToken: token,
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		secret, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Transcendence",
			AccountName: "user3@example.com",
		})
		if err != nil {
			t.Fatalf("failed to generate secret, err: %v", err)
		}
		preSecret := "pre-" + secret.Secret()
		user := db.User{
			Username:   "user3",
			Email:      "user3@example.com",
			TwoFAToken: &preSecret,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		setupToken, err := jwt.SignTwoFASetupToken(userService.Dep, user.ID, secret.Secret())
		if err != nil {
			t.Fatalf("failed to sign setup token, err: %v", err)
		}
		code, err := totp.GenerateCode(secret.Secret(), time.Now())
		if err != nil {
			t.Fatalf("failed to generate code, err: %v", err)
		}

		resp, err := userService.ConfirmTwoFaSetup(context.Background(), user.ID, &dto.TwoFAConfirmRequest{
			TwoFACode:  code,
			SetupToken: setupToken,
		})
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if resp.Token == "" {
			t.Fatalf("expected token in response")
		}

		modelUser, err := gorm.G[db.User](myDB).Where("id = ?", user.ID).First(context.Background())
		if err != nil {
			t.Fatalf("failed to query user, err: %v", err)
		}
		if modelUser.TwoFAToken == nil || *modelUser.TwoFAToken != secret.Secret() {
			t.Fatalf("expected two_fa_token to be enabled")
		}
	})
}

func TestDisableTwoFA(t *testing.T) {
	t.Run("oauth user", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		secret := "enabled-secret"
		user := db.User{
			Username:   "user1",
			Email:      "user1@example.com",
			TwoFAToken: &secret,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		_, err := userService.DisableTwoFA(context.Background(), user.ID, &dto.DisableTwoFARequest{
			Password: dto.Password{Password: "password"},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("2FA not enabled", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		hash, err := bcrypt.GenerateFromPassword([]byte("password"), 10)
		if err != nil {
			t.Fatalf("failed to hash password, err: %v", err)
		}
		passwordHash := string(hash)
		user := db.User{
			Username:     "user2",
			Email:        "user2@example.com",
			PasswordHash: &passwordHash,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		_, err = userService.DisableTwoFA(context.Background(), user.ID, &dto.DisableTwoFARequest{
			Password: dto.Password{Password: "password"},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("invalid password", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		hash, err := bcrypt.GenerateFromPassword([]byte("password"), 10)
		if err != nil {
			t.Fatalf("failed to hash password, err: %v", err)
		}
		passwordHash := string(hash)
		secret := "enabled-secret"
		user := db.User{
			Username:     "user3",
			Email:        "user3@example.com",
			PasswordHash: &passwordHash,
			TwoFAToken:   &secret,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		_, err = userService.DisableTwoFA(context.Background(), user.ID, &dto.DisableTwoFARequest{
			Password: dto.Password{Password: "wrong"},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 401 {
			t.Fatalf("expected status 401, got %d", authErr.Status)
		}
	})

	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		hash, err := bcrypt.GenerateFromPassword([]byte("password"), 10)
		if err != nil {
			t.Fatalf("failed to hash password, err: %v", err)
		}
		passwordHash := string(hash)
		secret := "enabled-secret"
		user := db.User{
			Username:     "user4",
			Email:        "user4@example.com",
			PasswordHash: &passwordHash,
			TwoFAToken:   &secret,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		resp, err := userService.DisableTwoFA(context.Background(), user.ID, &dto.DisableTwoFARequest{
			Password: dto.Password{Password: "password"},
		})
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if resp.Token == "" {
			t.Fatalf("expected token in response")
		}

		modelUser, err := gorm.G[db.User](myDB).Where("id = ?", user.ID).First(context.Background())
		if err != nil {
			t.Fatalf("failed to query user, err: %v", err)
		}
		if modelUser.TwoFAToken != nil {
			t.Fatalf("expected two_fa_token to be cleared")
		}
	})
}

func TestSubmitTwoFAChallenge(t *testing.T) {
	t.Run("invalid session token", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		_, err := userService.SubmitTwoFAChallenge(context.Background(), &dto.TwoFAChallengeRequest{
			TwoFACode:    "123456",
			SessionToken: "bad-token",
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("2FA not enabled", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{
			Username: "user1",
			Email:    "user1@example.com",
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		token, err := jwt.SignTwoFAToken(userService.Dep, user.ID)
		if err != nil {
			t.Fatalf("failed to sign session token, err: %v", err)
		}

		_, err = userService.SubmitTwoFAChallenge(context.Background(), &dto.TwoFAChallengeRequest{
			TwoFACode:    "123456",
			SessionToken: token,
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("invalid 2FA code", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		secret := "JBSWY3DPEHPK3PXP"
		user := db.User{
			Username:   "user2",
			Email:      "user2@example.com",
			TwoFAToken: &secret,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		token, err := jwt.SignTwoFAToken(userService.Dep, user.ID)
		if err != nil {
			t.Fatalf("failed to sign session token, err: %v", err)
		}

		_, err = userService.SubmitTwoFAChallenge(context.Background(), &dto.TwoFAChallengeRequest{
			TwoFACode:    "000000",
			SessionToken: token,
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 400 {
			t.Fatalf("expected status 400, got %d", authErr.Status)
		}
	})

	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		secret, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "Transcendence",
			AccountName: "user3@example.com",
		})
		if err != nil {
			t.Fatalf("failed to generate secret, err: %v", err)
		}
		secretStr := secret.Secret()
		user := db.User{
			Username:   "user3",
			Email:      "user3@example.com",
			TwoFAToken: &secretStr,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		token, err := jwt.SignTwoFAToken(userService.Dep, user.ID)
		if err != nil {
			t.Fatalf("failed to sign session token, err: %v", err)
		}
		code, err := totp.GenerateCode(secret.Secret(), time.Now())
		if err != nil {
			t.Fatalf("failed to generate code, err: %v", err)
		}

		resp, err := userService.SubmitTwoFAChallenge(context.Background(), &dto.TwoFAChallengeRequest{
			TwoFACode:    code,
			SessionToken: token,
		})
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if resp.Token == "" {
			t.Fatalf("expected token in response")
		}
	})
}
