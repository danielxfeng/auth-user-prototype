package service

import (
	"context"
	"testing"
	"time"

	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
)

func TestGetAllUsersLimitedInfo(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	// Create users
	_, _ = svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "u1"}, Email: "u1@e.com"},
		Password: dto.Password{Password: "p"},
	})
	_, _ = svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "u2"}, Email: "u2@e.com"},
		Password: dto.Password{Password: "p"},
	})

	t.Run("Success", func(t *testing.T) {
		users, err := svc.GetAllUsersLimitedInfo(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(users) < 2 {
			t.Errorf("expected at least 2 users, got %d", len(users))
		}
	})

	t.Run("DBError", func(t *testing.T) {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
		_, err := svc.GetAllUsersLimitedInfo(ctx)
		if err == nil {
			t.Error("expected error on closed db")
		}
	})
}

func TestAddNewFriend(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	u1, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "f1"}, Email: "f1@e.com"},
		Password: dto.Password{Password: "p"},
	})
	u2, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "f2"}, Email: "f2@e.com"},
		Password: dto.Password{Password: "p"},
	})

	cases := []struct {
		name          string
		userID        uint
		friendID      uint
		setup         func()
		wantErrStatus int
		checkFK       bool
	}{
		{"Success", u1.ID, u2.ID, nil, 0, false},
		{"AddSelf", u1.ID, u1.ID, nil, 400, false},
		{"DuplicateFriend", u1.ID, u2.ID, func() {
			_ = svc.AddNewFriend(ctx, u1.ID, &dto.AddNewFriendRequest{UserID: u2.ID})
		}, 409, false},
		{"UserNotFound", u1.ID, 999, nil, 404, true},
		{"DBError", u1.ID, u2.ID, func() {
			sqlDB, _ := db.DB()
			_ = sqlDB.Close()
		}, -1, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			err := svc.AddNewFriend(ctx, tc.userID, &dto.AddNewFriendRequest{UserID: tc.friendID})
			if tc.wantErrStatus == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.wantErrStatus == -1 {
				if err == nil {
					t.Error("expected error on closed db")
				}
				return
			}
			if err == nil && tc.checkFK {
				var count int64
				db.Model(&model.Friend{}).Where("user_id = ? AND friend_id = ?", u1.ID, tc.friendID).Count(&count)
				if count > 0 {
					t.Fatal("expected error, but friend was added despite FK violation")
				}
				return
			}
			requireAuthStatus(t, err, tc.wantErrStatus)
		})
	}
}

func TestGetUserFriends(t *testing.T) {
	db := setupTestDB(t.Name())
	svc := mustNewUserService(t, newTestDependency(db, nil))
	ctx := context.Background()

	u1, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "gf1"}, Email: "gf1@e.com"},
		Password: dto.Password{Password: "p"},
	})
	u2, _ := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "gf2"}, Email: "gf2@e.com"},
		Password: dto.Password{Password: "p"},
	})

	// Add friend
	_ = svc.AddNewFriend(ctx, u1.ID, &dto.AddNewFriendRequest{UserID: u2.ID})

	cases := []struct {
		name          string
		setup         func()
		wantOnline    bool
		wantErrStatus int
	}{
		{"Success", nil, false, 0},
		{"OnlineFriend", func() {
			db.Create(&model.HeartBeat{UserID: u2.ID, LastSeenAt: time.Now()})
		}, true, 0},
		{"DBError", func() {
			sqlDB, _ := db.DB()
			_ = sqlDB.Close()
		}, false, -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			friends, err := svc.GetUserFriends(ctx, u1.ID)
			if tc.wantErrStatus == -1 {
				if err == nil {
					t.Error("expected error on closed db")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(friends) != 1 {
				t.Errorf("expected 1 friend, got %d", len(friends))
			}
			if friends[0].ID != u2.ID {
				t.Errorf("expected friend ID %d, got %d", u2.ID, friends[0].ID)
			}
			if friends[0].Online != tc.wantOnline {
				t.Errorf("expected online=%v, got %v", tc.wantOnline, friends[0].Online)
			}
		})
	}
}
