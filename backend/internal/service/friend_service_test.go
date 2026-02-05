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
	"gorm.io/gorm"
)

func createFriend(t *testing.T, myDB *gorm.DB, userID, friendID uint) {
	t.Helper()

	friend := db.Friend{
		UserID:   userID,
		FriendID: friendID,
	}
	if err := gorm.G[db.Friend](myDB).Create(context.Background(), &friend); err != nil {
		t.Fatalf("failed to create friend, err: %v", err)
	}
}

func createHeartbeat(t *testing.T, myDB *gorm.DB, userID uint, lastSeen time.Time) {
	t.Helper()

	hb := db.HeartBeat{
		UserID:     userID,
		LastSeenAt: lastSeen,
	}
	if err := gorm.G[db.HeartBeat](myDB).Create(context.Background(), &hb); err != nil {
		t.Fatalf("failed to create heartbeat, err: %v", err)
	}
}

func TestGetAllUsersLimitedInfo(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		userService, _ := testutil.NewTestUserService(t)

		got, err := userService.GetAllUsersLimitedInfo(context.Background())
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected 0 users, got %d", len(got))
		}
	})

	t.Run("returns simple users", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		avatar1 := "https://example.com/1.png"
		avatar2 := "https://example.com/2.png"
		u1 := testutil.CreateUser(t, myDB, "alice", "alice@example.com", &avatar1)
		u2 := testutil.CreateUser(t, myDB, "bob", "bob@example.com", &avatar2)

		expected := map[uint]dto.SimpleUser{
			u1.ID: {ID: u1.ID, Username: "alice", Avatar: &avatar1},
			u2.ID: {ID: u2.ID, Username: "bob", Avatar: &avatar2},
		}

		got, err := userService.GetAllUsersLimitedInfo(context.Background())
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 users, got %d", len(got))
		}

		gotMap := make(map[uint]dto.SimpleUser, len(got))
		for _, u := range got {
			gotMap[u.ID] = u
		}

		for id, exp := range expected {
			gotUser, ok := gotMap[id]
			if !ok {
				t.Fatalf("missing user id %d", id)
			}
			if gotUser.Username != exp.Username {
				t.Fatalf("user %d username mismatch: expected %s, got %s", id, exp.Username, gotUser.Username)
			}
			if exp.Avatar == nil && gotUser.Avatar != nil {
				t.Fatalf("user %d expected nil avatar, got %v", id, gotUser.Avatar)
			}
			if exp.Avatar != nil && (gotUser.Avatar == nil || *gotUser.Avatar != *exp.Avatar) {
				t.Fatalf("user %d avatar mismatch: expected %v, got %v", id, exp.Avatar, gotUser.Avatar)
			}
		}
	})
}

func TestGetUserFriends(t *testing.T) {
	t.Run("no friends", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		u := testutil.CreateUser(t, myDB, "solo", "solo@example.com", nil)

		got, err := userService.GetUserFriends(context.Background(), u.ID)
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected 0 friends, got %d", len(got))
		}
	})

	t.Run("friends with online status", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := testutil.CreateUser(t, myDB, "owner", "owner@example.com", nil)
		friend1 := testutil.CreateUser(t, myDB, "friend1", "friend1@example.com", nil)
		friend2 := testutil.CreateUser(t, myDB, "friend2", "friend2@example.com", nil)

		createFriend(t, myDB, user.ID, friend1.ID)
		createFriend(t, myDB, user.ID, friend2.ID)

		createHeartbeat(t, myDB, friend2.ID, time.Now())

		expectedOnline := map[uint]bool{
			friend1.ID: false,
			friend2.ID: true,
		}

		got, err := userService.GetUserFriends(context.Background(), user.ID)
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 friends, got %d", len(got))
		}

		gotOnline := make(map[uint]bool, len(got))
		for _, f := range got {
			gotOnline[f.ID] = f.Online
		}

		for friendID, expected := range expectedOnline {
			isOnline, ok := gotOnline[friendID]
			if !ok {
				t.Fatalf("missing friend id %d", friendID)
			}
			if isOnline != expected {
				t.Fatalf("friend id %d online mismatch: expected %v, got %v", friendID, expected, isOnline)
			}
		}
	})
}

func TestAddNewFriend(t *testing.T) {
	t.Run("cannot add yourself", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := testutil.CreateUser(t, myDB, "self", "self@example.com", nil)

		err := userService.AddNewFriend(context.Background(), user.ID, &dto.AddNewFriendRequest{UserID: user.ID})
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

	t.Run("user not found", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := testutil.CreateUser(t, myDB, "owner", "owner@example.com", nil)

		err := userService.AddNewFriend(context.Background(), user.ID, &dto.AddNewFriendRequest{UserID: user.ID + 999})
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

	t.Run("friend already added", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := testutil.CreateUser(t, myDB, "owner", "owner@example.com", nil)
		friend := testutil.CreateUser(t, myDB, "friend", "friend@example.com", nil)
		createFriend(t, myDB, user.ID, friend.ID)

		err := userService.AddNewFriend(context.Background(), user.ID, &dto.AddNewFriendRequest{UserID: friend.ID})
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

	t.Run("success", func(t *testing.T) {
		userService, myDB := testutil.NewTestUserService(t)

		user := testutil.CreateUser(t, myDB, "owner", "owner@example.com", nil)
		friend := testutil.CreateUser(t, myDB, "friend", "friend@example.com", nil)

		err := userService.AddNewFriend(context.Background(), user.ID, &dto.AddNewFriendRequest{UserID: friend.ID})
		if err != nil {
			t.Fatalf("unexpected error, err: %v", err)
		}

		friendRecord, err := gorm.G[db.Friend](myDB).Where("user_id = ? AND friend_id = ?", user.ID, friend.ID).First(context.Background())
		if err != nil {
			t.Fatalf("failed to query friend record, err: %v", err)
		}
		if friendRecord.UserID != user.ID || friendRecord.FriendID != friend.ID {
			t.Fatalf("unexpected friend record values")
		}
	})
}
