package service_test

import (
	"context"
	"errors"
	"testing"

	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
	"github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestGetDependency(t *testing.T) {
	userService, _ := testutil.NewTestUserService(t)
	if userService.GetDependency() != userService.Dep {
		t.Fatalf("expected dependency to match")
	}
}

func TestCreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		req := &dto.CreateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "alice"},
				Email:    "alice@example.com",
			},
			Password: dto.Password{Password: "Password.777"},
		}

		resp, err := userService.CreateUser(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if resp.Username != "alice" {
			t.Fatalf("unexpected username: %s", resp.Username)
		}

		modelUser, err := gorm.G[db.User](myDB).Where("username = ?", "alice").First(context.Background())
		if err != nil {
			t.Fatalf("failed to query user, err: %v", err)
		}
		if modelUser.PasswordHash == nil || *modelUser.PasswordHash == "" {
			t.Fatalf("expected password hash to be set")
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		req1 := &dto.CreateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "dupuser"},
				Email:    "dup1@example.com",
			},
			Password: dto.Password{Password: "Password.777"},
		}
		_, err := userService.CreateUser(context.Background(), req1)
		if err != nil {
			t.Fatalf("unexpected error creating user, err: %v", err)
		}

		req2 := &dto.CreateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "dupuser"},
				Email:    "dup2@example.com",
			},
			Password: dto.Password{Password: "Password.777"},
		}
		_, err = userService.CreateUser(context.Background(), req2)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 409 {
			t.Fatalf("expected status 409, got %d", authErr.Status)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		req1 := &dto.CreateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "dupemail1"},
				Email:    "dup@example.com",
			},
			Password: dto.Password{Password: "Password.777"},
		}
		_, err := userService.CreateUser(context.Background(), req1)
		if err != nil {
			t.Fatalf("unexpected error creating user, err: %v", err)
		}

		req2 := &dto.CreateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "dupemail2"},
				Email:    "dup@example.com",
			},
			Password: dto.Password{Password: "Password.777"},
		}
		_, err = userService.CreateUser(context.Background(), req2)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 409 {
			t.Fatalf("expected status 409, got %d", authErr.Status)
		}
	})
}

func TestLoginUser(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		_, err := userService.LoginUser(context.Background(), &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "missing@example.com"},
			Password:   dto.Password{Password: "Password.777"},
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

	t.Run("invalid credentials", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		hash, err := bcrypt.GenerateFromPassword([]byte("Password.777"), 10)
		if err != nil {
			t.Fatalf("failed to hash password, err: %v", err)
		}
		passwordHash := string(hash)
		user := db.User{
			Username:     "alice",
			Email:        "alice@example.com",
			PasswordHash: &passwordHash,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		_, err = userService.LoginUser(context.Background(), &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "alice"},
			Password:   dto.Password{Password: "Wrong.777"},
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

	t.Run("login by username", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		hash, err := bcrypt.GenerateFromPassword([]byte("Password.777"), 10)
		if err != nil {
			t.Fatalf("failed to hash password, err: %v", err)
		}
		passwordHash := string(hash)
		user := db.User{
			Username:     "alice",
			Email:        "alice@example.com",
			PasswordHash: &passwordHash,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		result, err := userService.LoginUser(context.Background(), &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "alice"},
			Password:   dto.Password{Password: "Password.777"},
		})
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if result.User == nil || result.User.Token == "" {
			t.Fatalf("expected user token")
		}
	})

	t.Run("login by email", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		hash, err := bcrypt.GenerateFromPassword([]byte("Password.777"), 10)
		if err != nil {
			t.Fatalf("failed to hash password, err: %v", err)
		}
		passwordHash := string(hash)
		user := db.User{
			Username:     "alice",
			Email:        "alice@example.com",
			PasswordHash: &passwordHash,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		result, err := userService.LoginUser(context.Background(), &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "alice@example.com"},
			Password:   dto.Password{Password: "Password.777"},
		})
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if result.User == nil || result.User.Token == "" {
			t.Fatalf("expected user token")
		}
	})

	t.Run("2FA pending", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		hash, err := bcrypt.GenerateFromPassword([]byte("Password.777"), 10)
		if err != nil {
			t.Fatalf("failed to hash password, err: %v", err)
		}
		passwordHash := string(hash)
		secret := "enabled-secret"
		user := db.User{
			Username:     "alice",
			Email:        "alice@example.com",
			PasswordHash: &passwordHash,
			TwoFAToken:   &secret,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		result, err := userService.LoginUser(context.Background(), &dto.LoginUserRequest{
			Identifier: dto.Identifier{Identifier: "alice"},
			Password:   dto.Password{Password: "Password.777"},
		})
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if result.TwoFAPending == nil || result.TwoFAPending.SessionToken == "" {
			t.Fatalf("expected 2FA pending session token")
		}
	})
}

func TestGetUserByID(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		_, err := userService.GetUserByID(context.Background(), 9999)
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

	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{
			Username: "bob",
			Email:    "bob@example.com",
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		resp, err := userService.GetUserByID(context.Background(), user.ID)
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if resp.Username != "bob" {
			t.Fatalf("unexpected username: %s", resp.Username)
		}
	})
}

func TestUpdateUserPassword(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		_, err := userService.UpdateUserPassword(context.Background(), 9999, &dto.UpdateUserPasswordRequest{
			OldPassword: dto.OldPassword{OldPassword: "Password.777"},
			NewPassword: dto.NewPassword{NewPassword: "Password.888"},
		})
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

	t.Run("oauth user", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{
			Username: "oauth",
			Email:    "oauth@example.com",
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		_, err := userService.UpdateUserPassword(context.Background(), user.ID, &dto.UpdateUserPasswordRequest{
			OldPassword: dto.OldPassword{OldPassword: "Password.777"},
			NewPassword: dto.NewPassword{NewPassword: "Password.888"},
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

	t.Run("invalid old password", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		hash, err := bcrypt.GenerateFromPassword([]byte("Password.777"), 10)
		if err != nil {
			t.Fatalf("failed to hash password, err: %v", err)
		}
		passwordHash := string(hash)
		user := db.User{
			Username:     "user1",
			Email:        "user1@example.com",
			PasswordHash: &passwordHash,
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		_, err = userService.UpdateUserPassword(context.Background(), user.ID, &dto.UpdateUserPasswordRequest{
			OldPassword: dto.OldPassword{OldPassword: "Wrong.777"},
			NewPassword: dto.NewPassword{NewPassword: "Password.888"},
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

		hash, err := bcrypt.GenerateFromPassword([]byte("Password.777"), 10)
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

		resp, err := userService.UpdateUserPassword(context.Background(), user.ID, &dto.UpdateUserPasswordRequest{
			OldPassword: dto.OldPassword{OldPassword: "Password.777"},
			NewPassword: dto.NewPassword{NewPassword: "Password.888"},
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
		if modelUser.PasswordHash == nil {
			t.Fatalf("expected password hash to be set")
		}
		if bcrypt.CompareHashAndPassword([]byte(*modelUser.PasswordHash), []byte("Password.888")) != nil {
			t.Fatalf("expected password hash to match new password")
		}
	})
}

func TestUpdateUserProfile(t *testing.T) {
	t.Run("user not found", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		_, err := userService.UpdateUserProfile(context.Background(), 9999, &dto.UpdateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "user1"},
				Email:    "user1@example.com",
			},
		})
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

	t.Run("duplicate username", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user1 := db.User{Username: "user1", Email: "user1@example.com"}
		user2 := db.User{Username: "user2", Email: "user2@example.com"}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user1); err != nil {
			t.Fatalf("failed to create user1, err: %v", err)
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user2); err != nil {
			t.Fatalf("failed to create user2, err: %v", err)
		}

		_, err := userService.UpdateUserProfile(context.Background(), user2.ID, &dto.UpdateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "user1"},
				Email:    "user2@example.com",
			},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 409 {
			t.Fatalf("expected status 409, got %d", authErr.Status)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user1 := db.User{Username: "user1", Email: "user1@example.com"}
		user2 := db.User{Username: "user2", Email: "user2@example.com"}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user1); err != nil {
			t.Fatalf("failed to create user1, err: %v", err)
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user2); err != nil {
			t.Fatalf("failed to create user2, err: %v", err)
		}

		_, err := userService.UpdateUserProfile(context.Background(), user2.ID, &dto.UpdateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "user2"},
				Email:    "user1@example.com",
			},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		var authErr *authError.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("expected auth error, got: %v", err)
		}
		if authErr.Status != 409 {
			t.Fatalf("expected status 409, got %d", authErr.Status)
		}
	})

	t.Run("avatar cleared by empty string", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		avatar := "https://example.com/a.png"
		user := db.User{Username: "user1", Email: "user1@example.com", Avatar: &avatar}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		blank := "   "
		_, err := userService.UpdateUserProfile(context.Background(), user.ID, &dto.UpdateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "user1"},
				Email:    "user1@example.com",
				Avatar:   &blank,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}

		modelUser, err := gorm.G[db.User](myDB).Where("id = ?", user.ID).First(context.Background())
		if err != nil {
			t.Fatalf("failed to query user, err: %v", err)
		}
		if modelUser.Avatar != nil {
			t.Fatalf("expected avatar to be cleared")
		}
	})

	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{Username: "user1", Email: "user1@example.com"}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		newAvatar := "https://example.com/new.png"
		_, err := userService.UpdateUserProfile(context.Background(), user.ID, &dto.UpdateUserRequest{
			User: dto.User{
				UserName: dto.UserName{Username: "user1-updated"},
				Email:    "user1-updated@example.com",
				Avatar:   &newAvatar,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}

		modelUser, err := gorm.G[db.User](myDB).Where("id = ?", user.ID).First(context.Background())
		if err != nil {
			t.Fatalf("failed to query user, err: %v", err)
		}
		if modelUser.Username != "user1-updated" || modelUser.Email != "user1-updated@example.com" {
			t.Fatalf("expected user profile to be updated")
		}
		if modelUser.Avatar == nil || *modelUser.Avatar != newAvatar {
			t.Fatalf("expected avatar to be updated")
		}
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{Username: "user1", Email: "user1@example.com"}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		if err := userService.DeleteUser(context.Background(), user.ID); err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}

		_, err := gorm.G[db.User](myDB).Where("id = ?", user.ID).First(context.Background())
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected user to be deleted")
		}
	})

	t.Run("missing user", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		if err := userService.DeleteUser(context.Background(), 9999); err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
	})
}

func TestLogoutUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{Username: "user1", Email: "user1@example.com"}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		token := db.Token{
			UserID: user.ID,
			Token:  "token-1",
		}
		if err := gorm.G[db.Token](myDB).Create(context.Background(), &token); err != nil {
			t.Fatalf("failed to create token, err: %v", err)
		}

		if err := userService.LogoutUser(context.Background(), user.ID); err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}

		_, err := gorm.G[db.Token](myDB).Where("token = ?", "token-1").First(context.Background())
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected token to be deleted")
		}
	})

	t.Run("no tokens", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{Username: "user2", Email: "user2@example.com"}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		if err := userService.LogoutUser(context.Background(), user.ID); err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
	})
}

func TestValidateUserToken(t *testing.T) {
	t.Run("invalid token", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := db.User{Username: "user1", Email: "user1@example.com"}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		err := userService.ValidateUserToken(context.Background(), "nope", user.ID)
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

	t.Run("token does not match user", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user1 := db.User{Username: "user1", Email: "user1@example.com"}
		user2 := db.User{Username: "user2", Email: "user2@example.com"}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user1); err != nil {
			t.Fatalf("failed to create user1, err: %v", err)
		}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user2); err != nil {
			t.Fatalf("failed to create user2, err: %v", err)
		}

		token := db.Token{
			UserID: user1.ID,
			Token:  "token-1",
		}
		if err := gorm.G[db.Token](myDB).Create(context.Background(), &token); err != nil {
			t.Fatalf("failed to create token, err: %v", err)
		}

		err := userService.ValidateUserToken(context.Background(), "token-1", user2.ID)
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

		user := db.User{Username: "user1", Email: "user1@example.com"}
		if err := gorm.G[db.User](myDB).Create(context.Background(), &user); err != nil {
			t.Fatalf("failed to create user, err: %v", err)
		}

		token := db.Token{
			UserID: user.ID,
			Token:  "token-1",
		}
		if err := gorm.G[db.Token](myDB).Create(context.Background(), &token); err != nil {
			t.Fatalf("failed to create token, err: %v", err)
		}

		err := userService.ValidateUserToken(context.Background(), "token-1", user.ID)
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
	})
}
