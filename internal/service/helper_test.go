package service

import (
	"context"
	"testing"
	"time"

	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
)

func TestHelperFunctions(t *testing.T) {
	t.Run("isTwoFAEnabled", func(t *testing.T) {
		token := "pre-secret"
		if isTwoFAEnabled(&token) {
			t.Error("expected false for pre- prefix")
		}
		
		token = "secret"
		if !isTwoFAEnabled(&token) {
			t.Error("expected true for valid secret")
		}

		token = ""
		if isTwoFAEnabled(&token) {
			t.Error("expected false for empty token")
		}

		if isTwoFAEnabled(nil) {
			t.Error("expected false for nil token")
		}
	})

	t.Run("userToUserWithTokenResponse", func(t *testing.T) {
		token := "secret"
		user := &model.User{
			Username: "u",
			TwoFAToken: &token,
		}
		resp := userToUserWithTokenResponse(user, "jwt")
		if !resp.TwoFA {
			t.Error("expected 2FA true")
		}
		if resp.Token != "jwt" {
			t.Error("expected token match")
		}
	})

	t.Run("OnlineStatusChecker", func(t *testing.T) {
		hbs := []model.HeartBeat{
			{UserID: 1},
		}
		checker := newOnlineStatusChecker(hbs)
		if !checker.isOnline(1) {
			t.Error("expected 1 online")
		}
		if checker.isOnline(2) {
			t.Error("expected 2 offline")
		}
	})
	
	t.Run("UpdateHeartBeat", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := NewUserService(db)
		
		// Create user first to satisfy FK
		svc.CreateUser(context.Background(), &dto.CreateUserRequest{
			User: dto.User{UserName: dto.UserName{Username: "hb"}, Email: "hb@e.com"},
			Password: dto.Password{Password: "p"},
		})

		// Create heartbeat entry
		svc.updateHeartBeat(1)
		
		// Wait for goroutine
		time.Sleep(100 * time.Millisecond)
		
		var hb model.HeartBeat
		if err := db.Where("user_id = ?", 1).First(&hb).Error; err != nil {
			t.Fatalf("expected heartbeat created: %v", err)
		}
	})

	t.Run("IssueNewTokenForUser", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := NewUserService(db)
		
		// Create user first
		svc.CreateUser(context.Background(), &dto.CreateUserRequest{
			User: dto.User{UserName: dto.UserName{Username: "issue"}, Email: "issue@e.com"},
			Password: dto.Password{Password: "p"},
		})

		token, err := svc.issueNewTokenForUser(context.Background(), 1, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token == "" {
			t.Error("expected token")
		}
		
		// Allow async heartbeat to finish
		time.Sleep(200 * time.Millisecond)

		// Revoke old tokens
		svc.issueNewTokenForUser(context.Background(), 1, true)
		var count int64
		db.Model(&model.Token{}).Where("user_id = ?", 1).Count(&count)
		if count != 1 {
			t.Errorf("expected 1 token, got %d", count)
		}
	})
	
	t.Run("IssueNewTokenForUser_DBError", func(t *testing.T) {
		db := setupTestDB(t.Name())
		svc := NewUserService(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
		
		_, err := svc.issueNewTokenForUser(context.Background(), 1, true)
		if err == nil {
			t.Error("expected error on closed db")
		}
	})
}
