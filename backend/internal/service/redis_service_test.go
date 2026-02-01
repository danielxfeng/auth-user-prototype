package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"github.com/redis/go-redis/v9"
)

func withRedisTestExpiries(cfg *config.Config, userTTLSeconds int, absoluteTTLSeconds int) {
	cfg.UserTokenExpiry = userTTLSeconds
	cfg.UserTokenAbsoluteExpiry = absoluteTTLSeconds
}

func TestRedisTokenLifecycle(t *testing.T) {
	db := setupTestDB(t.Name())
	cfg := testutil.NewTestConfig()
	withRedisTestExpiries(cfg, 10, 30)
	mr, redisClient, cleanupRedis := setupTestRedis(t, cfg)
	defer cleanupRedis()

	svc := mustNewUserService(t, newTestDependencyWithConfig(cfg, db, redisClient))
	ctx := context.Background()

	userResp, err := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "redisuser"}, Email: "redis@example.com"},
		Password: dto.Password{Password: "password123"},
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	token, err := svc.issueNewTokenForUser(ctx, userResp.ID, false)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	key := buildTokenKey(userResp.ID, token)
	if !mr.Exists(key) {
		t.Fatalf("expected redis token key to exist: %s", key)
	}

	// Drive time close to expiry, then validate and ensure TTL slides forward.
	mr.FastForward(9 * time.Second)
	ttlBefore := mr.TTL(key)
	if ttlBefore <= 0 {
		t.Fatalf("expected TTL before validation to be positive, got %v", ttlBefore)
	}

	if err := svc.ValidateUserToken(ctx, token, userResp.ID); err != nil {
		t.Fatalf("expected token to validate, got %v", err)
	}

	ttlAfter := mr.TTL(key)
	if ttlAfter < 8*time.Second {
		t.Fatalf("expected sliding TTL refresh, got %v", ttlAfter)
	}

	// Logout should revoke all redis tokens for the user.
	if err := svc.LogoutUser(ctx, userResp.ID); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	if mr.Exists(key) {
		t.Fatal("expected redis token key to be deleted on logout")
	}

	err = svc.ValidateUserToken(ctx, token, userResp.ID)
	if err == nil {
		t.Fatal("expected token to be invalid after logout")
	}
	var authErr *authError.AuthError
	if !strings.Contains(err.Error(), "invalid token") || !errors.As(err, &authErr) {
		t.Fatalf("expected auth error for invalid token, got %v", err)
	}
}

func TestRedisHeartbeatOnlineStatusAndCleanup(t *testing.T) {
	db := setupTestDB(t.Name())
	cfg := testutil.NewTestConfig()
	_, redisClient, cleanupRedis := setupTestRedis(t, cfg)
	defer cleanupRedis()

	svc := mustNewUserService(t, newTestDependencyWithConfig(cfg, db, redisClient))
	ctx := context.Background()

	u1, err := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "hb1"}, Email: "hb1@example.com"},
		Password: dto.Password{Password: "password123"},
	})
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	_, err = svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "hb2"}, Email: "hb2@example.com"},
		Password: dto.Password{Password: "password123"},
	})
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	svc.updateHeartBeat(u1.ID)
	time.Sleep(100 * time.Millisecond)

	onlineNow, err := svc.getOnlineStatus(ctx)
	if err != nil {
		t.Fatalf("getOnlineStatus failed: %v", err)
	}

	checkerNow := newOnlineStatusChecker(onlineNow)
	if !checkerNow.isOnline(u1.ID) {
		t.Fatal("expected user1 to be online after heartbeat")
	}

	// Force the heartbeat score to be old, then ensure cleanup happens.
	oldScore := float64(time.Now().Add(-3 * time.Minute).Unix())
	if err := redisClient.ZAdd(ctx, HeartBeatPrefix, redis.Z{Score: oldScore, Member: u1.ID}).Err(); err != nil {
		t.Fatalf("failed to set old heartbeat score: %v", err)
	}

	onlineLater, err := svc.getOnlineStatus(ctx)
	if err != nil {
		t.Fatalf("getOnlineStatus later failed: %v", err)
	}

	checkerLater := newOnlineStatusChecker(onlineLater)
	if checkerLater.isOnline(u1.ID) {
		t.Fatal("expected user1 to be offline after expiration window")
	}

	// Cleanup should have removed the expired heartbeat entry.
	time.Sleep(100 * time.Millisecond)
	if _, err := redisClient.ZScore(ctx, HeartBeatPrefix, fmt.Sprint(u1.ID)).Result(); err == nil {
		t.Fatal("expected expired heartbeat to be removed from redis")
	}
}

func TestRedisLoginUpdatesHeartbeat(t *testing.T) {
	db := setupTestDB(t.Name())
	cfg := testutil.NewTestConfig()
	_, redisClient, cleanupRedis := setupTestRedis(t, cfg)
	defer cleanupRedis()

	svc := mustNewUserService(t, newTestDependencyWithConfig(cfg, db, redisClient))
	ctx := context.Background()

	created, err := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "loginhb"}, Email: "loginhb@example.com"},
		Password: dto.Password{Password: "password123"},
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	userID := created.ID

	res, err := svc.LoginUser(ctx, &dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "loginhb"},
		Password:   dto.Password{Password: "password123"},
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if res.User == nil || res.User.Token == "" {
		t.Fatal("expected login to issue a valid token")
	}

	time.Sleep(100 * time.Millisecond)

	score, err := redisClient.ZScore(ctx, HeartBeatPrefix, fmt.Sprint(userID)).Result()
	if err != nil {
		t.Fatalf("expected heartbeat entry for user, got error: %v", err)
	}
	now := time.Now().Unix()
	if int64(score) < now-5 {
		t.Fatalf("expected recent heartbeat score, got %v (now=%d)", score, now)
	}
}

func TestRedisLogoutRevokesAllTokens(t *testing.T) {
	db := setupTestDB(t.Name())
	cfg := testutil.NewTestConfig()
	mr, redisClient, cleanupRedis := setupTestRedis(t, cfg)
	defer cleanupRedis()

	svc := mustNewUserService(t, newTestDependencyWithConfig(cfg, db, redisClient))
	ctx := context.Background()

	userResp, err := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "logoutmulti"}, Email: "logoutmulti@example.com"},
		Password: dto.Password{Password: "password123"},
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	token1, err := svc.issueNewTokenForUser(ctx, userResp.ID, false)
	if err != nil {
		t.Fatalf("failed to issue token1: %v", err)
	}
	token2, err := svc.issueNewTokenForUser(ctx, userResp.ID, false)
	if err != nil {
		t.Fatalf("failed to issue token2: %v", err)
	}

	key1 := buildTokenKey(userResp.ID, token1)
	key2 := buildTokenKey(userResp.ID, token2)
	if !mr.Exists(key1) || !mr.Exists(key2) {
		t.Fatalf("expected both redis token keys to exist: %s, %s", key1, key2)
	}

	if err := svc.LogoutUser(ctx, userResp.ID); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	if mr.Exists(key1) || mr.Exists(key2) {
		t.Fatal("expected redis token keys to be deleted on logout")
	}

	if err := svc.ValidateUserToken(ctx, token1, userResp.ID); err == nil {
		t.Fatal("expected token1 to be invalid after logout")
	}
	if err := svc.ValidateUserToken(ctx, token2, userResp.ID); err == nil {
		t.Fatal("expected token2 to be invalid after logout")
	}
}

func TestRedisDeleteUserRevokesAllTokens(t *testing.T) {
	db := setupTestDB(t.Name())
	cfg := testutil.NewTestConfig()
	mr, redisClient, cleanupRedis := setupTestRedis(t, cfg)
	defer cleanupRedis()

	svc := mustNewUserService(t, newTestDependencyWithConfig(cfg, db, redisClient))
	ctx := context.Background()

	userResp, err := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "delredis"}, Email: "delredis@example.com"},
		Password: dto.Password{Password: "password123"},
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	token1, err := svc.issueNewTokenForUser(ctx, userResp.ID, false)
	if err != nil {
		t.Fatalf("failed to issue token1: %v", err)
	}
	token2, err := svc.issueNewTokenForUser(ctx, userResp.ID, false)
	if err != nil {
		t.Fatalf("failed to issue token2: %v", err)
	}

	key1 := buildTokenKey(userResp.ID, token1)
	key2 := buildTokenKey(userResp.ID, token2)
	if !mr.Exists(key1) || !mr.Exists(key2) {
		t.Fatalf("expected both redis token keys to exist: %s, %s", key1, key2)
	}

	if err := svc.DeleteUser(ctx, userResp.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if mr.Exists(key1) || mr.Exists(key2) {
		t.Fatal("expected redis token keys to be deleted on user deletion")
	}

	if err := svc.ValidateUserToken(ctx, token1, userResp.ID); err == nil {
		t.Fatal("expected token1 to be invalid after delete")
	}
	if err := svc.ValidateUserToken(ctx, token2, userResp.ID); err == nil {
		t.Fatal("expected token2 to be invalid after delete")
	}
}

func TestRedisUpdatePasswordRevokesOldTokens(t *testing.T) {
	db := setupTestDB(t.Name())
	cfg := testutil.NewTestConfig()
	mr, redisClient, cleanupRedis := setupTestRedis(t, cfg)
	defer cleanupRedis()

	svc := mustNewUserService(t, newTestDependencyWithConfig(cfg, db, redisClient))
	ctx := context.Background()

	userResp, err := svc.CreateUser(ctx, &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "pwredis"}, Email: "pwredis@example.com"},
		Password: dto.Password{Password: "oldpass"},
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	token1, err := svc.issueNewTokenForUser(ctx, userResp.ID, false)
	if err != nil {
		t.Fatalf("failed to issue token1: %v", err)
	}
	token2, err := svc.issueNewTokenForUser(ctx, userResp.ID, false)
	if err != nil {
		t.Fatalf("failed to issue token2: %v", err)
	}

	key1 := buildTokenKey(userResp.ID, token1)
	key2 := buildTokenKey(userResp.ID, token2)
	if !mr.Exists(key1) || !mr.Exists(key2) {
		t.Fatalf("expected both redis token keys to exist: %s, %s", key1, key2)
	}

	updateReq := &dto.UpdateUserPasswordRequest{
		OldPassword: dto.OldPassword{OldPassword: "oldpass"},
		NewPassword: dto.NewPassword{NewPassword: "newpass"},
	}
	resp, err := svc.UpdateUserPassword(ctx, userResp.ID, updateReq)
	if err != nil {
		t.Fatalf("update password failed: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("expected new token from password update")
	}

	if mr.Exists(key1) || mr.Exists(key2) {
		t.Fatal("expected old redis token keys to be deleted on password change")
	}

	if err := svc.ValidateUserToken(ctx, token1, userResp.ID); err == nil {
		t.Fatal("expected token1 to be invalid after password change")
	}
	if err := svc.ValidateUserToken(ctx, token2, userResp.ID); err == nil {
		t.Fatal("expected token2 to be invalid after password change")
	}
	if err := svc.ValidateUserToken(ctx, resp.Token, userResp.ID); err != nil {
		t.Fatalf("expected new token to be valid after password change, got %v", err)
	}
}
