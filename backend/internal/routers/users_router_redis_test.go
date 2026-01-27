package routers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	db "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
)

func setupUsersRouterTestRedis(t *testing.T) (*gin.Engine, *miniredis.Miniredis, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	util.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	prevCfg := config.Cfg
	config.Cfg = &config.Config{
		JwtSecret:               "test-secret",
		UserTokenExpiry:         60,
		UserTokenAbsoluteExpiry: 600,
		TwoFaTokenExpiry:        3600,
		OauthStateTokenExpiry:   3600,
		GoogleClientId:          "test-client",
		GoogleRedirectUri:       "http://localhost/cb",
		FrontendUrl:             "http://localhost:3000",
		IsRedisEnabled:          true,
	}
	dto.InitValidator()

	// DB setup matches existing patterns.
	dbName := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_busy_timeout=5000&_foreign_keys=on"
	var err error
	db.DB, err = gorm.Open(sqlite.Open(dbName), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	db.DB.Exec("PRAGMA foreign_keys = ON")

	err = db.DB.AutoMigrate(&db.User{}, &db.Friend{}, &db.Token{}, &db.HeartBeat{})
	if err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	// Redis setup.
	mr := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	db.Redis = redisClient
	config.Cfg.RedisURL = "redis://" + mr.Addr()

	router := gin.New()
	UsersRouter(router.Group("/users"))

	if db.DB != nil {
		sqlDB, _ := db.DB.DB()
		if sqlDB != nil {
			sqlDB.SetMaxOpenConns(1)
		}
	}

	cleanup := func() {
		config.Cfg = prevCfg
		if db.Redis != nil {
			_ = db.Redis.Close()
			db.Redis = nil
		}
		mr.Close()
		if db.DB != nil {
			sqlDB, _ := db.DB.DB()
			if sqlDB != nil {
				_ = sqlDB.Close()
			}
			db.DB = nil
		}
	}

	return router, mr, cleanup
}

func TestUsersRouter_Redis_LoginValidateLogout(t *testing.T) {
	router, mr, cleanup := setupUsersRouterTestRedis(t)
	defer cleanup()

	// Create user
	createReq := dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "redisrouter"}, Email: "redisrouter@example.com"},
		Password: dto.Password{Password: "password123"},
	}
	createBody, _ := json.Marshal(createReq)
	createResp := httptest.NewRecorder()
	createHTTP := httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBuffer(createBody))
	createHTTP.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createResp, createHTTP)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create, got %d. Body: %s", createResp.Code, createResp.Body.String())
	}

	// Login user
	loginReq := dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "redisrouter"},
		Password:   dto.Password{Password: "password123"},
	}
	loginBody, _ := json.Marshal(loginReq)
	loginResp := httptest.NewRecorder()
	loginHTTP := httptest.NewRequest(http.MethodPost, "/users/loginByIdentifier", bytes.NewBuffer(loginBody))
	loginHTTP.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(loginResp, loginHTTP)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d. Body: %s", loginResp.Code, loginResp.Body.String())
	}

	var loginUser dto.UserWithTokenResponse
	_ = json.Unmarshal(loginResp.Body.Bytes(), &loginUser)
	if loginUser.Token == "" || loginUser.ID == 0 {
		t.Fatalf("expected login token and id, got: %+v", loginUser)
	}

	// Ensure token is stored in Redis (by key prefix).
	keys := mr.Keys()
	wantPrefix := "user_token:" + strconv.FormatUint(uint64(loginUser.ID), 10) + ":"
	foundTokenKey := false
	for _, k := range keys {
		if strings.HasPrefix(k, wantPrefix) {
			foundTokenKey = true
			break
		}
	}
	if !foundTokenKey {
		t.Fatalf("expected redis token key with prefix %q, keys: %v", wantPrefix, keys)
	}

	// Login should update heartbeat in Redis.
	time.Sleep(100 * time.Millisecond)
	score, err := db.Redis.ZScore(context.Background(), "heartbeat:", strconv.FormatUint(uint64(loginUser.ID), 10)).Result()
	if err != nil {
		t.Fatalf("expected heartbeat entry after login, got error: %v", err)
	}
	if int64(score) < time.Now().Unix()-5 {
		t.Fatalf("expected recent heartbeat score after login, got %v", score)
	}

	// Validate should succeed
	validateResp := httptest.NewRecorder()
	validateHTTP := httptest.NewRequest(http.MethodPost, "/users/validate", nil)
	validateHTTP.Header.Set("Authorization", "Bearer "+loginUser.Token)
	router.ServeHTTP(validateResp, validateHTTP)
	if validateResp.Code != http.StatusOK {
		t.Fatalf("expected 200 on validate, got %d. Body: %s", validateResp.Code, validateResp.Body.String())
	}

	// Logout should revoke redis tokens
	logoutResp := httptest.NewRecorder()
	logoutHTTP := httptest.NewRequest(http.MethodDelete, "/users/logout", nil)
	logoutHTTP.Header.Set("Authorization", "Bearer "+loginUser.Token)
	router.ServeHTTP(logoutResp, logoutHTTP)
	if logoutResp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on logout, got %d. Body: %s", logoutResp.Code, logoutResp.Body.String())
	}

	// Validate again should fail
	validateAfterResp := httptest.NewRecorder()
	validateAfterHTTP := httptest.NewRequest(http.MethodPost, "/users/validate", nil)
	validateAfterHTTP.Header.Set("Authorization", "Bearer "+loginUser.Token)
	router.ServeHTTP(validateAfterResp, validateAfterHTTP)
	if validateAfterResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on validate after logout, got %d. Body: %s", validateAfterResp.Code, validateAfterResp.Body.String())
	}
}
