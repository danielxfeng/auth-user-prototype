package service

import (
	"context"
	"strings"
	"testing"

	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
)

func requireAuthStatus(t *testing.T, err error, status int) {
	t.Helper()
	authErr, ok := err.(*middleware.AuthError)
	if !ok || authErr.Status != status {
		t.Fatalf("expected %d error, got %v", status, err)
	}
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	cases := []struct {
		name          string
		req           *dto.CreateUserRequest
		setup         func()
		wantErrStatus int
	}{
		{
			name: "Success",
			req: &dto.CreateUserRequest{
				User: dto.User{
					UserName: dto.UserName{Username: "testuser"},
					Email:    "test@example.com",
				},
				Password: dto.Password{Password: "password123"},
			},
		},
		{
			name: "DuplicateUsername",
			req: &dto.CreateUserRequest{
				User: dto.User{
					UserName: dto.UserName{Username: "testuser"},
					Email:    "other@example.com",
				},
				Password: dto.Password{Password: "password123"},
			},
			setup: func() {
				_, _ = svc.CreateUser(ctx, &dto.CreateUserRequest{
					User: dto.User{
						UserName: dto.UserName{Username: "testuser"},
						Email:    "test@example.com",
					},
					Password: dto.Password{Password: "password123"},
				})
			},
			wantErrStatus: 409,
		},
		{
			name: "DuplicateEmail",
			req: &dto.CreateUserRequest{
				User: dto.User{
					UserName: dto.UserName{Username: "otheruser"},
					Email:    "test@example.com",
				},
				Password: dto.Password{Password: "password123"},
			},
			setup: func() {
				_, _ = svc.CreateUser(ctx, &dto.CreateUserRequest{
					User: dto.User{
						UserName: dto.UserName{Username: "seeduser"},
						Email:    "test@example.com",
					},
					Password: dto.Password{Password: "password123"},
				})
			},
			wantErrStatus: 409,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			resp, err := svc.CreateUser(ctx, tc.req)
			if tc.wantErrStatus == 0 {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if resp.Username != tc.req.Username {
					t.Errorf("expected username %s, got %s", tc.req.Username, resp.Username)
				}
				if resp.Email != tc.req.Email {
					t.Errorf("expected email %s, got %s", tc.req.Email, resp.Email)
				}
				if resp.ID == 0 {
					t.Error("expected valid ID")
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			requireAuthStatus(t, err, tc.wantErrStatus)
		})
	}
}

func TestLoginUser(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	// Setup user
	createReq := &dto.CreateUserRequest{
		User: dto.User{
			UserName: dto.UserName{Username: "loginuser"},
			Email:    "login@example.com",
		},
		Password: dto.Password{Password: "password123"},
	}
	_, _ = svc.CreateUser(ctx, createReq)

	t.Run("SuccessUsername", func(t *testing.T) {
		req := &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "loginuser"},
			Password:   dto.Password{Password: "password123"},
		}

		res, err := svc.LoginUser(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.User == nil || res.User.Token == "" {
			t.Error("expected user with token")
		}
	})

	t.Run("SuccessEmail", func(t *testing.T) {
		req := &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "login@example.com"},
			Password:   dto.Password{Password: "password123"},
		}

		res, err := svc.LoginUser(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.User == nil || res.User.Token == "" {
			t.Error("expected user with token")
		}
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		req := &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "loginuser"},
			Password:   dto.Password{Password: "wrongpass"},
		}

		_, err := svc.LoginUser(ctx, req)
		if err == nil {
			t.Fatal("expected error")
		}
		authErr, ok := err.(*middleware.AuthError)
		if !ok || authErr.Status != 401 {
			t.Errorf("expected 401 error, got %v", err)
		}
	})

	t.Run("UserNotFound", func(t *testing.T) {
		req := &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "nonexistent"},
			Password:   dto.Password{Password: "password123"},
		}

		_, err := svc.LoginUser(ctx, req)
		if err == nil {
			t.Fatal("expected error")
		}
		authErr, ok := err.(*middleware.AuthError)
		if !ok || authErr.Status != 401 {
			t.Errorf("expected 401 error, got %v", err)
		}
	})

	t.Run("2FARequired", func(t *testing.T) {
		// Enable 2FA for user
		user, _ := svc.GetUserByID(ctx, 1) // First user created (loginuser)
		_, _ = svc.StartTwoFaSetup(ctx, user.ID)
		// We need to confirm it properly, but we can hack it for this test
		db.Model(&model.User{}).Where("id = ?", user.ID).Update("two_fa_token", "secret")

		req := &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "loginuser"},
			Password:   dto.Password{Password: "password123"},
		}

		res, err := svc.LoginUser(ctx, req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.User != nil {
			t.Error("expected no user token when 2FA required")
		}
		if res.TwoFAPending == nil {
			t.Error("expected 2FA pending response")
		}
		if res.TwoFAPending.Message != "2FA_REQUIRED" {
			t.Errorf("expected message 2FA_REQUIRED, got %s", res.TwoFAPending.Message)
		}
	})

	t.Run("OAuthUser", func(t *testing.T) {
		// Create oauth user
		oauthUser := dto.GoogleUserData{ID: "login_oauth", Email: "login_oauth@e.com"}
		user, _ := svc.createNewUserFromGoogleInfo(ctx, &oauthUser, false)

		req := &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: user.Username},
			Password:   dto.Password{Password: "any"},
		}

		_, err := svc.LoginUser(ctx, req)
		if err == nil {
			t.Fatal("expected error")
		}
		authErr, ok := err.(*middleware.AuthError)
		if !ok || authErr.Status != 401 {
			t.Errorf("expected 401 error, got %v", err)
		}
	})

	t.Run("InvalidHash", func(t *testing.T) {
		// Manually create user with invalid hash
		_, _ = svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "badhash"}, Email: "badhash@e.com"},
			Password: dto.Password{Password: "p"},
		})
		badHash := "invalid_hash"
		db.Model(&model.User{}).Where("username = ?", "badhash").Update("password_hash", badHash)

		req := &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "badhash"},
			Password:   dto.Password{Password: "p"},
		}

		_, err := svc.LoginUser(ctx, req)
		if err == nil {
			t.Fatal("expected error")
		}
		// Should return raw error, not AuthError
		if _, ok := err.(*middleware.AuthError); ok {
			t.Error("expected raw error for invalid hash")
		}
	})
}

func TestGetUserByID(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User: dto.User{
			UserName: dto.UserName{Username: "getuser"},
			Email:    "get@example.com",
		},
		Password: dto.Password{Password: "pass"},
	})

	cases := []struct {
		name          string
		userID        uint
		wantErrStatus int
	}{
		{"Success", u.ID, 0},
		{"NotFound", 9999, 404},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := svc.GetUserByID(ctx, tc.userID)
			if tc.wantErrStatus == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.ID != u.ID {
					t.Errorf("want ID %d, got %d", u.ID, got.ID)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			requireAuthStatus(t, err, tc.wantErrStatus)
		})
	}
}

func TestUpdateUserPassword(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User: dto.User{
			UserName: dto.UserName{Username: "passupdate"},
			Email:    "pass@example.com",
		},
		Password: dto.Password{Password: "oldpass"},
	})

	t.Run("Success", func(t *testing.T) {
		req := &dto.UpdateUserPasswordRequest{
			OldPassword: dto.OldPassword{OldPassword: "oldpass"},
			NewPassword: dto.NewPassword{NewPassword: "newpass"},
		}

		resp, err := svc.UpdateUserPassword(ctx, u.ID, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Token == "" {
			t.Error("expected new token")
		}

		loginReq := &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "passupdate"},
			Password:   dto.Password{Password: "newpass"},
		}
		if _, err := svc.LoginUser(ctx, loginReq); err != nil {
			t.Error("failed to login with new password")
		}
	})

	errorCases := []struct {
		name          string
		setup         func() uint
		req           *dto.UpdateUserPasswordRequest
		wantErrStatus int
	}{
		{
			name: "InvalidOldPassword",
			setup: func() uint {
				return u.ID
			},
			req: &dto.UpdateUserPasswordRequest{
				OldPassword: dto.OldPassword{OldPassword: "wrongold"},
				NewPassword: dto.NewPassword{NewPassword: "newpass2"},
			},
			wantErrStatus: 401,
		},
		{
			name: "OAuthUser",
			setup: func() uint {
				oauthUser := dto.GoogleUserData{ID: "passoauth", Email: "passoauth@e.com"}
				user, _ := svc.createNewUserFromGoogleInfo(ctx, &oauthUser, false)
				return user.ID
			},
			req: &dto.UpdateUserPasswordRequest{
				OldPassword: dto.OldPassword{OldPassword: "any"},
				NewPassword: dto.NewPassword{NewPassword: "new"},
			},
			wantErrStatus: 400,
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			userID := tc.setup()
			_, err := svc.UpdateUserPassword(ctx, userID, tc.req)
			if err == nil {
				t.Fatal("expected error")
			}
			requireAuthStatus(t, err, tc.wantErrStatus)
		})
	}

	t.Run("InvalidHash", func(t *testing.T) {
		_, _ = svc.CreateUser(ctx, &dto.CreateUserRequest{
			User:     dto.User{UserName: dto.UserName{Username: "badhash2"}, Email: "badhash2@e.com"},
			Password: dto.Password{Password: "password123"},
		})
		badHash := "invalid_hash"
		var user model.User
		db.Where("username = ?", "badhash2").First(&user)
		db.Model(&user).Update("password_hash", badHash)

		req := &dto.UpdateUserPasswordRequest{
			OldPassword: dto.OldPassword{OldPassword: "password123"},
			NewPassword: dto.NewPassword{NewPassword: "new"},
		}

		_, err := svc.UpdateUserPassword(ctx, user.ID, req)
		if err == nil {
			t.Fatal("expected error")
		}
		if _, ok := err.(*middleware.AuthError); ok {
			t.Error("expected raw error")
		}
	})
}

func TestUpdateUserProfile(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User: dto.User{
			UserName: dto.UserName{Username: "updateprofile"},
			Email:    "update@example.com",
		},
		Password: dto.Password{Password: "pass"},
	})

	cases := []struct {
		name          string
		setup         func()
		req           *dto.UpdateUserRequest
		wantErrStatus int
	}{
		{
			name: "Success",
			req: &dto.UpdateUserRequest{
				User: dto.User{
					UserName: dto.UserName{Username: "newname"},
					Email:    "new@example.com",
					Avatar:   func() *string { v := "new_avatar.png"; return &v }(),
				},
			},
		},
		{
			name: "Duplicate",
			setup: func() {
				_, _ = svc.CreateUser(ctx, &dto.CreateUserRequest{
					User:     dto.User{UserName: dto.UserName{Username: "other"}, Email: "other@e.com"},
					Password: dto.Password{Password: "password123"},
				})
			},
			req: &dto.UpdateUserRequest{
				User: dto.User{
					UserName: dto.UserName{Username: "other"},
					Email:    "new@example.com",
				},
			},
			wantErrStatus: 409,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			got, err := svc.UpdateUserProfile(ctx, u.ID, tc.req)
			if tc.wantErrStatus == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.Username != tc.req.Username {
					t.Errorf("want username %s, got %s", tc.req.Username, got.Username)
				}
				if got.Email != tc.req.Email {
					t.Errorf("want email %s, got %s", tc.req.Email, got.Email)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error for duplicate")
			}
			requireAuthStatus(t, err, tc.wantErrStatus)
		})
	}
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	u, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User: dto.User{
			UserName: dto.UserName{Username: "deleteuser"},
			Email:    "del@example.com",
		},
		Password: dto.Password{Password: "pass"},
	})

	t.Run("Success", func(t *testing.T) {
		err := svc.DeleteUser(ctx, u.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = svc.GetUserByID(ctx, u.ID)
		if err == nil {
			t.Error("expected user to be deleted")
		}
	})
}

func TestValidateUserToken(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	createReq := &dto.CreateUserRequest{
		User: dto.User{
			UserName: dto.UserName{Username: "tokenuser"},
			Email:    "token@example.com",
		},
		Password: dto.Password{Password: "pass"},
	}
	_, _ = svc.CreateUser(ctx, createReq)

	loginRes, _ := svc.LoginUser(ctx, &dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "tokenuser"},
		Password:   dto.Password{Password: "pass"},
	})
	token := loginRes.User.Token
	userID := loginRes.User.ID

	t.Run("Success", func(t *testing.T) {
		err := svc.ValidateUserToken(ctx, token, userID)
		if err != nil {
			t.Errorf("expected token to be valid, got %v", err)
		}
	})

	t.Run("InvalidToken", func(t *testing.T) {
		err := svc.ValidateUserToken(ctx, "invalidtoken", userID)
		if err == nil {
			t.Error("expected error for invalid token")
		}
	})

	t.Run("TokenMismatchUser", func(t *testing.T) {
		u2, err := svc.CreateUser(ctx, &dto.CreateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "user2"},
				Email:    "u2@ex.com",
			},
			Password: dto.Password{Password: "pass"},
		})
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		err = svc.ValidateUserToken(ctx, token, u2.ID)
		if err == nil {
			t.Error("expected error for token mismatch")
		}
		if !strings.Contains(err.Error(), "token does not match user") {
			t.Errorf("expected mismatch error, got %v", err)
		}
	})
}

func TestLogoutUser(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	createReq := &dto.CreateUserRequest{
		User: dto.User{
			UserName: dto.UserName{Username: "logoutuser"},
			Email:    "logout@example.com",
		},
		Password: dto.Password{Password: "pass"},
	}
	_, _ = svc.CreateUser(ctx, createReq)

	loginRes, _ := svc.LoginUser(ctx, &dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "logoutuser"},
		Password:   dto.Password{Password: "pass"},
	})
	token := loginRes.User.Token
	userID := loginRes.User.ID

	err := svc.LogoutUser(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = svc.ValidateUserToken(ctx, token, userID)
	if err == nil {
		t.Error("expected token to be invalid after logout")
	}
}

func TestDBErrors(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		run  func(svc *UserService) error
	}{
		{
			name: "CreateUser",
			run: func(svc *UserService) error {
				req := &dto.CreateUserRequest{
					User:     dto.User{UserName: dto.UserName{Username: "db1"}, Email: "db1@e.com"},
					Password: dto.Password{Password: "password123"},
				}
				_, err := svc.CreateUser(ctx, req)
				return err
			},
		},
		{
			name: "LoginUser",
			run: func(svc *UserService) error {
				req := &dto.LoginUserRequest{
					Identifier: dto.Identifier{Identifier: "db1"},
					Password:   dto.Password{Password: "password123"},
				}
				_, err := svc.LoginUser(ctx, req)
				return err
			},
		},
		{
			name: "GetUserByID",
			run: func(svc *UserService) error {
				_, err := svc.GetUserByID(ctx, 1)
				return err
			},
		},
		{
			name: "UpdateUserPassword",
			run: func(svc *UserService) error {
				req := &dto.UpdateUserPasswordRequest{
					OldPassword: dto.OldPassword{OldPassword: "password123"},
					NewPassword: dto.NewPassword{NewPassword: "password456"},
				}
				_, err := svc.UpdateUserPassword(ctx, 1, req)
				return err
			},
		},
		{
			name: "UpdateUserProfile",
			run: func(svc *UserService) error {
				req := &dto.UpdateUserRequest{
					User: dto.User{UserName: dto.UserName{Username: "n"}, Email: "n@e.com"},
				}
				_, err := svc.UpdateUserProfile(ctx, 1, req)
				return err
			},
		},
		{
			name: "DeleteUser",
			run: func(svc *UserService) error {
				return svc.DeleteUser(ctx, 1)
			},
		},
		{
			name: "ValidateUserToken",
			run: func(svc *UserService) error {
				return svc.ValidateUserToken(ctx, "token", 1)
			},
		},
		{
			name: "LogoutUser",
			run: func(svc *UserService) error {
				return svc.LogoutUser(ctx, 1)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupTestDB(t.Name())
			svc := mustNewUserService(t, newTestDependency(db, nil))
			sqlDB, _ := db.DB()
			_ = sqlDB.Close()

			if err := tc.run(svc); err == nil {
				t.Error("expected db error")
			}
		})
	}
}
