package service

import (
	"context"
	"testing"
	"time"

	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
	"github.com/pquerna/otp/totp"
)

func TestTwoFASetupAndConfirm(t *testing.T) {
	ctx := context.Background()

	t.Run("StartSetup_Success", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "u1"}, Email: "u1@e.com"},
			Password: dto.Password{Password: "p"},
		})

		resp, err := svc.StartTwoFaSetup(ctx, u.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.TwoFASecret == "" {
			t.Error("expected secret")
		}
	})

	t.Run("ConfirmSetup_Success", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "u2"}, Email: "u2@e.com"},
			Password: dto.Password{Password: "p"},
		})

		resp, _ := svc.StartTwoFaSetup(ctx, u.ID)
		code, err := totp.GenerateCode(resp.TwoFASecret, time.Now())
		if err != nil {
			t.Fatalf("failed to generate code: %v", err)
		}

		req := &dto.TwoFAConfirmRequest{
			SetupToken: resp.SetupToken,
			TwoFACode:  code,
		}

		res, err := svc.ConfirmTwoFaSetup(ctx, u.ID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !res.TwoFA {
			t.Error("expected 2FA to be enabled")
		}
	})

	t.Run("StartSetup_AlreadyEnabled", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "u3"}, Email: "u3@e.com"},
			Password: dto.Password{Password: "p"},
		})
		resp, _ := svc.StartTwoFaSetup(ctx, u.ID)
		code, _ := totp.GenerateCode(resp.TwoFASecret, time.Now())
		_, _ = svc.ConfirmTwoFaSetup(ctx, u.ID, &dto.TwoFAConfirmRequest{
			SetupToken: resp.SetupToken,
			TwoFACode:  code,
		})

		_, err := svc.StartTwoFaSetup(ctx, u.ID)
		if err == nil {
			t.Fatal("expected error")
		}
		authErr, ok := err.(*authError.AuthError)
		if !ok || authErr.Status != 400 {
			t.Errorf("expected 400 error, got %v", err)
		}
	})

	t.Run("StartSetup_OAuthUser", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		// Mock OAuth user
		oauthUser := dto.GoogleUserData{
			ID:    "oauth123",
			Email: "oauth@test.com",
		}
		user, err := svc.createNewUserFromGoogleInfo(ctx, &oauthUser, false)
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		_, err = svc.StartTwoFaSetup(ctx, user.ID)
		if err == nil {
			t.Fatal("expected error for oauth user")
		}
		authErr, ok := err.(*authError.AuthError)
		if !ok || authErr.Status != 400 {
			t.Errorf("expected 400 error, got %v", err)
		}
	})

	t.Run("StartSetup_DBError", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "u4"}, Email: "u4@e.com"},
			Password: dto.Password{Password: "p"},
		})

		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		_, err := svc.StartTwoFaSetup(ctx, u.ID)
		if err == nil {
			t.Error("expected error on closed db")
		}
	})
}

func TestConfirmTwoFaSetup_Errors(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "c"}, Email: "c@e.com"},
		Password: dto.Password{Password: "p"},
	})
	resp, _ := svc.StartTwoFaSetup(ctx, u.ID)
	code, _ := totp.GenerateCode(resp.TwoFASecret, time.Now())

	t.Run("InvalidToken", func(t *testing.T) {
		req := &dto.TwoFAConfirmRequest{
			SetupToken: "invalid",
			TwoFACode:  code,
		}
		_, err := svc.ConfirmTwoFaSetup(ctx, u.ID, req)
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})

	t.Run("UserMismatch", func(t *testing.T) {
		// Create setup token for another user
		u2, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "c2"}, Email: "c2@e.com"},
			Password: dto.Password{Password: "p"},
		})
		resp2, _ := svc.StartTwoFaSetup(ctx, u2.ID)

		req := &dto.TwoFAConfirmRequest{
			SetupToken: resp2.SetupToken,
			TwoFACode:  code,
		}
		_, err := svc.ConfirmTwoFaSetup(ctx, u.ID, req) // Wrong user ID
		if err == nil {
			t.Error("expected error for user mismatch")
		}
	})

	t.Run("WrongTokenType", func(t *testing.T) {
		token, _ := jwt.SignUserToken(svc.Dep, u.ID)
		req := &dto.TwoFAConfirmRequest{
			SetupToken: token,
			TwoFACode:  code,
		}
		_, err := svc.ConfirmTwoFaSetup(ctx, u.ID, req)
		if err == nil {
			t.Error("expected error for wrong token type")
		}
	})

	t.Run("DBError", func(t *testing.T) {
		req := &dto.TwoFAConfirmRequest{
			SetupToken: resp.SetupToken,
			TwoFACode:  code,
		}

		sqlDB, _ := db.DB()
		_ = sqlDB.Close()

		_, err := svc.ConfirmTwoFaSetup(ctx, u.ID, req)
		if err == nil {
			t.Error("expected error on closed db")
		}
	})

	t.Run("NotInitiated", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		// User with no 2FA token
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "ni"}, Email: "ni@e.com"},
			Password: dto.Password{Password: "p"},
		})

		// Create a valid setup token manually
		setupToken, _ := jwt.SignTwoFASetupToken(svc.Dep, u.ID, "secret")
		code, _ := totp.GenerateCode("secret", time.Now())

		req := &dto.TwoFAConfirmRequest{
			SetupToken: setupToken,
			TwoFACode:  code,
		}

		_, err := svc.ConfirmTwoFaSetup(ctx, u.ID, req)
		if err == nil {
			t.Error("expected error for not initiated")
		}
	})
}

func TestTwoFAChallenge(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "ch1"}, Email: "ch1@e.com"},
			Password: dto.Password{Password: "p"},
		})
		setupResp, _ := svc.StartTwoFaSetup(ctx, u.ID)
		code, _ := totp.GenerateCode(setupResp.TwoFASecret, time.Now())
		_, _ = svc.ConfirmTwoFaSetup(ctx, u.ID, &dto.TwoFAConfirmRequest{SetupToken: setupResp.SetupToken, TwoFACode: code})

		loginResp, _ := svc.LoginUser(ctx, &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "ch1"},
			Password:   dto.Password{Password: "p"},
		})
		sessionToken := loginResp.TwoFAPending.SessionToken

		code, _ = totp.GenerateCode(setupResp.TwoFASecret, time.Now())
		req := &dto.TwoFAChallengeRequest{
			SessionToken: sessionToken,
			TwoFACode:    code,
		}

		resp, err := svc.SubmitTwoFAChallenge(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Token == "" {
			t.Error("expected valid user token")
		}
	})

	t.Run("InvalidCode", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "ch2"}, Email: "ch2@e.com"},
			Password: dto.Password{Password: "p"},
		})
		setupResp, _ := svc.StartTwoFaSetup(ctx, u.ID)
		code, _ := totp.GenerateCode(setupResp.TwoFASecret, time.Now())
		_, _ = svc.ConfirmTwoFaSetup(ctx, u.ID, &dto.TwoFAConfirmRequest{SetupToken: setupResp.SetupToken, TwoFACode: code})

		loginResp, _ := svc.LoginUser(ctx, &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "ch2"},
			Password:   dto.Password{Password: "p"},
		})
		sessionToken := loginResp.TwoFAPending.SessionToken

		req := &dto.TwoFAChallengeRequest{
			SessionToken: sessionToken,
			TwoFACode:    "000000",
		}

		_, err := svc.SubmitTwoFAChallenge(ctx, req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("NotEnabled", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "chne"}, Email: "chne@e.com"},
			Password: dto.Password{Password: "p"},
		})
		// Do NOT enable 2FA

		// Create session token manually
		sessionToken, _ := jwt.SignTwoFAToken(svc.Dep, u.ID)

		req := &dto.TwoFAChallengeRequest{
			SessionToken: sessionToken,
			TwoFACode:    "000000",
		}

		_, err := svc.SubmitTwoFAChallenge(ctx, req)
		if err == nil {
			t.Fatal("expected error for not enabled")
		}
	})

	t.Run("DBError", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "ch3"}, Email: "ch3@e.com"},
			Password: dto.Password{Password: "p"},
		})
		setupResp, _ := svc.StartTwoFaSetup(ctx, u.ID)
		code, _ := totp.GenerateCode(setupResp.TwoFASecret, time.Now())
		_, _ = svc.ConfirmTwoFaSetup(ctx, u.ID, &dto.TwoFAConfirmRequest{SetupToken: setupResp.SetupToken, TwoFACode: code})

		loginResp, _ := svc.LoginUser(ctx, &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "ch3"},
			Password:   dto.Password{Password: "p"},
		})
		sessionToken := loginResp.TwoFAPending.SessionToken

		req := &dto.TwoFAChallengeRequest{
			SessionToken: sessionToken,
			TwoFACode:    code,
		}

		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		_, err := svc.SubmitTwoFAChallenge(ctx, req)
		if err == nil {
			t.Error("expected error on closed db")
		}
	})
}

func TestDisableTwoFA(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "dis1"}, Email: "dis1@e.com"},
			Password: dto.Password{Password: "p"},
		})
		setupResp, _ := svc.StartTwoFaSetup(ctx, u.ID)
		code, _ := totp.GenerateCode(setupResp.TwoFASecret, time.Now())
		_, _ = svc.ConfirmTwoFaSetup(ctx, u.ID, &dto.TwoFAConfirmRequest{SetupToken: setupResp.SetupToken, TwoFACode: code})
		req := &dto.DisableTwoFARequest{
			Password: dto.Password{Password: "p"},
		}

		resp, err := svc.DisableTwoFA(ctx, u.ID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.TwoFA {
			t.Error("expected 2FA to be disabled")
		}
	})

	t.Run("AlreadyDisabled", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "dis2"}, Email: "dis2@e.com"},
			Password: dto.Password{Password: "p"},
		})

		req := &dto.DisableTwoFARequest{
			Password: dto.Password{Password: "p"},
		}
		_, err := svc.DisableTwoFA(ctx, u.ID, req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("OAuthUser", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		// Mock OAuth user
		oauthUser := dto.GoogleUserData{
			ID:    "oauth456",
			Email: "oauth2@test.com",
		}
		user, _ := svc.createNewUserFromGoogleInfo(ctx, &oauthUser, false)

		req := &dto.DisableTwoFARequest{
			Password: dto.Password{Password: "any"},
		}
		_, err := svc.DisableTwoFA(ctx, user.ID, req)
		if err == nil {
			t.Fatal("expected error for oauth user")
		}
		authErr, ok := err.(*authError.AuthError)
		if !ok || authErr.Status != 400 {
			t.Errorf("expected 400 error, got %v", err)
		}
	})

	t.Run("DBError", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "dis3"}, Email: "dis3@e.com"},
			Password: dto.Password{Password: "p"},
		})

		sqlDB, _ := db.DB()
		_ = sqlDB.Close()

		req := &dto.DisableTwoFARequest{
			Password: dto.Password{Password: "p"},
		}
		_, err := svc.DisableTwoFA(ctx, u.ID, req)
		if err == nil {
			t.Error("expected error on closed db")
		}
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := mustNewUserService(t, newTestDependency(db, nil))
		u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "disinv"}, Email: "disinv@e.com"},
			Password: dto.Password{Password: "correct"},
		})

		// Enable 2FA manually
		setupResp, _ := svc.StartTwoFaSetup(ctx, u.ID)
		code, _ := totp.GenerateCode(setupResp.TwoFASecret, time.Now())
		_, _ = svc.ConfirmTwoFaSetup(ctx, u.ID, &dto.TwoFAConfirmRequest{SetupToken: setupResp.SetupToken, TwoFACode: code})
		req := &dto.DisableTwoFARequest{
			Password: dto.Password{Password: "wrong"},
		}
		_, err := svc.DisableTwoFA(ctx, u.ID, req)
		if err == nil {
			t.Fatal("expected error for invalid password")
		}
		authErr, ok := err.(*authError.AuthError)
		if !ok || authErr.Status != 401 {
			t.Errorf("expected 401 error, got %v", err)
		}
	})
}
