package routers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/auth/credentials/idtoken"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/service"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

type usersRouterEnv struct {
	router  *gin.Engine
	dep     *dependency.Dependency
	mr      *miniredis.Miniredis
	cleanup func()
}

func setupUsersRouterTest(t *testing.T, useRedis bool) *usersRouterEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	logger := testutil.NewTestLogger()
	cfg := testutil.NewTestConfig()
	cfg.JwtSecret = "test-secret"
	cfg.UserTokenExpiry = 3600
	cfg.UserTokenAbsoluteExpiry = 600
	cfg.TwoFaTokenExpiry = 3600
	cfg.OauthStateTokenExpiry = 3600
	cfg.GoogleClientId = "test-client"
	cfg.GoogleRedirectUri = "http://localhost/cb"
	cfg.FrontendUrl = "http://localhost:3000"

	if useRedis {
		cfg.IsRedisEnabled = true
	}

	dto.InitValidator()

	dbName := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_busy_timeout=5000&_foreign_keys=on"
	dbConn, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}

	dbConn.Exec("PRAGMA foreign_keys = ON")
	if err := dbConn.AutoMigrate(&model.User{}, &model.Friend{}, &model.Token{}, &model.HeartBeat{}); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	var mr *miniredis.Miniredis
	var redisClient *redis.Client
	if useRedis {
		mr = miniredis.RunT(t)
		redisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
		cfg.RedisURL = "redis://" + mr.Addr()
	}

	dep := dependency.NewDependency(cfg, dbConn, redisClient, logger)
	router := gin.New()
	UsersRouter(router.Group("/users"), dep)

	if sqlDB, err := dbConn.DB(); err == nil && sqlDB != nil {
		sqlDB.SetMaxOpenConns(1)
	}

	cleanup := func() {
		if redisClient != nil {
			_ = redisClient.Close()
		}
		if mr != nil {
			mr.Close()
		}
		if sqlDB, err := dbConn.DB(); err == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	}

	return &usersRouterEnv{
		router:  router,
		dep:     dep,
		mr:      mr,
		cleanup: cleanup,
	}
}

func signUserToken(t *testing.T, dep *dependency.Dependency, userID uint) string {
	t.Helper()
	token, err := jwt.SignUserToken(dep, userID)
	if err != nil {
		t.Fatalf("failed to sign user token: %v", err)
	}
	return token
}

func addUserToken(t *testing.T, dep *dependency.Dependency, userID uint) string {
	t.Helper()
	token := signUserToken(t, dep, userID)
	if err := dep.DB.Create(&model.Token{UserID: userID, Token: token}).Error; err != nil {
		t.Fatalf("failed to insert token: %v", err)
	}
	return token
}

func createUser(t *testing.T, dep *dependency.Dependency, username, email, password string) *dto.UserWithoutTokenResponse {
	t.Helper()
	svc := service.NewUserService(dep)
	user, err := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: username}, Email: email},
		Password: dto.Password{Password: password},
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func TestUsersRouter_CreateUser(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	reqBody := dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "newuser"}, Email: "new@example.com"},
		Password: dto.Password{Password: "password123"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var user dto.UserWithoutTokenResponse
	_ = json.Unmarshal(resp.Body.Bytes(), &user)
	if user.Username != "newuser" {
		t.Errorf("expected username newuser, got %s", user.Username)
	}
}

func TestUsersRouter_CreateUser_Failures(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"InvalidBody", `{"username": "u"}`, http.StatusBadRequest},
		{"DuplicateUser", "duplicate", http.StatusConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.body
			if tc.name == "DuplicateUser" {
				validReq := dto.CreateUserRequest{
					User:     dto.User{UserName: dto.UserName{Username: "dupuser"}, Email: "dup@e.com"},
					Password: dto.Password{Password: "pass123"},
				}
				payload, _ := json.Marshal(validReq)
				env.router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBuffer(payload)))
				body = string(payload)
			}

			req := httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			env.router.ServeHTTP(resp, req)

			if resp.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.Code)
			}
		})
	}
}

func TestUsersRouter_LoginUser(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	createReq := dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "loginuser"}, Email: "login@example.com"},
		Password: dto.Password{Password: "password123"},
	}
	createBody, _ := json.Marshal(createReq)

	cReq := httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBuffer(createBody))
	cReq.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(httptest.NewRecorder(), cReq)

	loginReq := dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "loginuser"},
		Password:   dto.Password{Password: "password123"},
	}
	body, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/users/loginByIdentifier", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var res dto.UserWithTokenResponse
	_ = json.Unmarshal(resp.Body.Bytes(), &res)
	if res.Token == "" {
		t.Errorf("expected token in response. Body: %s", resp.Body.String())
	}
}

func TestUsersRouter_LoginUser_Failures(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	cases := []struct {
		name       string
		body       string
		setup      func()
		wantStatus int
	}{
		{"InvalidBody", `{}`, nil, http.StatusBadRequest},
		{"UserNotFound", "missing", nil, http.StatusUnauthorized},
		{"WrongPassword", "wrong", func() {
			_ = createUser(t, env.dep, "loginfail", "fail@e.com", "correct123")
		}, http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}

			var payload []byte
			switch tc.body {
			case "missing":
				loginReq := dto.LoginUserRequest{Identifier: dto.Identifier{Identifier: "missing"}, Password: dto.Password{Password: "pass123"}}
				payload, _ = json.Marshal(loginReq)
			case "wrong":
				loginReq := dto.LoginUserRequest{Identifier: dto.Identifier{Identifier: "loginfail"}, Password: dto.Password{Password: "wrong123"}}
				payload, _ = json.Marshal(loginReq)
			default:
				payload = []byte(tc.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/users/loginByIdentifier", bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			env.router.ServeHTTP(resp, req)

			if resp.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.Code)
			}
		})
	}
}

func TestUsersRouter_GetProfile(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	user := model.User{Username: "profileuser", Email: "profile@example.com"}
	env.dep.DB.Create(&user)
	tokenStr := addUserToken(t, env.dep, user.ID)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var res dto.UserWithoutTokenResponse
	_ = json.Unmarshal(resp.Body.Bytes(), &res)
	if res.Username != "profileuser" {
		t.Errorf("expected username profileuser, got %s", res.Username)
	}
}

func TestUsersRouter_Unauthorized(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	resp := httptest.NewRecorder()

	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}
}

func TestUsersRouter_UpdateUserProfile(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	user := model.User{Username: "u", Email: "u@e.com"}
	env.dep.DB.Create(&user)
	tokenStr := addUserToken(t, env.dep, user.ID)

	newAvatar := "http://pic.com/1.png"
	reqBody := dto.UpdateUserRequest{User: dto.User{UserName: dto.UserName{Username: "newname"}, Email: "new@e.com", Avatar: &newAvatar}}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var res dto.UserWithoutTokenResponse
	_ = json.Unmarshal(resp.Body.Bytes(), &res)
	if res.Username != "newname" {
		t.Errorf("expected new username, got %s", res.Username)
	}
}

func TestUsersRouter_UpdateUser_Failures(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	svc := service.NewUserService(env.dep)
	u, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "u1"}, Email: "u1@e.com"},
		Password: dto.Password{Password: "pass123"},
	})
	token := addUserToken(t, env.dep, u.ID)

	_, _ = svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "u2"}, Email: "u2@e.com"},
		Password: dto.Password{Password: "pass123"},
	})

	cases := []struct {
		name       string
		method     string
		path       string
		body       any
		wantStatus int
	}{
		{"DuplicateProfile", http.MethodPut, "/users/me", dto.UpdateUserRequest{User: dto.User{UserName: dto.UserName{Username: "update_u2"}, Email: "u2@e.com"}}, http.StatusConflict},
		{"WrongOldPassword", http.MethodPut, "/users/password", dto.UpdateUserPasswordRequest{OldPassword: dto.OldPassword{OldPassword: "wrong123"}, NewPassword: dto.NewPassword{NewPassword: "newpass"}}, http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBuffer(payload))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			env.router.ServeHTTP(resp, req)
			if resp.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.Code)
			}
		})
	}
}

func TestUsersRouter_UpdateUserPassword(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	svc := service.NewUserService(env.dep)
	userResp, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "pw"}, Email: "pw@e.com"},
		Password: dto.Password{Password: "oldpass"},
	})
	tokenStr := addUserToken(t, env.dep, userResp.ID)

	reqBody := dto.UpdateUserPasswordRequest{
		OldPassword: dto.OldPassword{OldPassword: "oldpass"},
		NewPassword: dto.NewPassword{NewPassword: "newpass"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/users/password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}
}

func TestUsersRouter_DeleteUser(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	user := model.User{Username: "del", Email: "del@e.com"}
	env.dep.DB.Create(&user)
	tokenStr := addUserToken(t, env.dep, user.ID)

	time.Sleep(500 * time.Millisecond)

	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.Code)
	}
}

func TestUsersRouter_GetUsersWithLimitedInfo(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	user := model.User{Username: "list", Email: "list@e.com"}
	env.dep.DB.Create(&user)
	tokenStr := addUserToken(t, env.dep, user.ID)

	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestUsersRouter_ValidateUser(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	user := model.User{Username: "val", Email: "val@e.com"}
	env.dep.DB.Create(&user)
	tokenStr := addUserToken(t, env.dep, user.ID)

	req := httptest.NewRequest(http.MethodPost, "/users/validate", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var res dto.UserValidationResponse
	_ = json.Unmarshal(resp.Body.Bytes(), &res)
	if res.UserID != user.ID {
		t.Errorf("expected userID %d, got %d", user.ID, res.UserID)
	}
}

func TestUsersRouter_Friends(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	svc := service.NewUserService(env.dep)
	u1 := createUser(t, env.dep, "f1", "f1@e.com", "pass123")
	u2 := createUser(t, env.dep, "f2", "f2@e.com", "pass123")
	_ = svc

	tokenStr := addUserToken(t, env.dep, u1.ID)

	reqBody := dto.AddNewFriendRequest{UserID: u2.ID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/users/friends", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()
	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/users/friends", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp = httptest.NewRecorder()
	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	var friends []dto.FriendResponse
	_ = json.Unmarshal(resp.Body.Bytes(), &friends)
	if len(friends) != 1 || friends[0].ID != u2.ID {
		t.Error("expected friend f2")
	}
}

func TestUsersRouter_Friends_Failures(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	svc := service.NewUserService(env.dep)
	u1 := createUser(t, env.dep, "f1", "f1@e.com", "pass123")
	u2 := createUser(t, env.dep, "f2", "f2@e.com", "pass123")
	token := addUserToken(t, env.dep, u1.ID)

	cases := []struct {
		name       string
		payload    dto.AddNewFriendRequest
		setup      func()
		wantStatus int
	}{
		{"AddSelf", dto.AddNewFriendRequest{UserID: u1.ID}, nil, http.StatusBadRequest},
		{"AddMissing", dto.AddNewFriendRequest{UserID: 999}, nil, http.StatusNotFound},
		{"Duplicate", dto.AddNewFriendRequest{UserID: u2.ID}, func() {
			_ = svc.AddNewFriend(context.Background(), u1.ID, &dto.AddNewFriendRequest{UserID: u2.ID})
		}, http.StatusConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
				if tc.name == "Duplicate" {
					time.Sleep(200 * time.Millisecond)
				}
			}
			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/users/friends", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			env.router.ServeHTTP(resp, req)
			if resp.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.Code)
			}
		})
	}
}

func TestUsersRouter_2FA(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	user := createUser(t, env.dep, "2fa", "2fa@e.com", "pass123")
	tokenStr := addUserToken(t, env.dep, user.ID)

	req := httptest.NewRequest(http.MethodPost, "/users/2fa/setup", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()
	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", resp.Code)
	}
	var setupRes dto.TwoFASetupResponse
	_ = json.Unmarshal(resp.Body.Bytes(), &setupRes)

	time.Sleep(200 * time.Millisecond)

	code, _ := totp.GenerateCode(setupRes.TwoFASecret, time.Now())
	confirmBody, _ := json.Marshal(dto.TwoFAConfirmRequest{SetupToken: setupRes.SetupToken, TwoFACode: code})
	req = httptest.NewRequest(http.MethodPost, "/users/2fa/confirm", bytes.NewBuffer(confirmBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp = httptest.NewRecorder()
	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("confirm failed: %d", resp.Code)
	}

	sessionToken, _ := jwt.SignTwoFAToken(env.dep, user.ID)
	code, _ = totp.GenerateCode(setupRes.TwoFASecret, time.Now())
	challengeBody, _ := json.Marshal(dto.TwoFAChallengeRequest{SessionToken: sessionToken, TwoFACode: code})
	req = httptest.NewRequest(http.MethodPost, "/users/2fa", bytes.NewBuffer(challengeBody))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("challenge failed: %d body: %s", resp.Code, resp.Body.String())
	}

	var userRes dto.UserWithTokenResponse
	_ = json.Unmarshal(resp.Body.Bytes(), &userRes)
	tokenStr = userRes.Token

	time.Sleep(200 * time.Millisecond)

	disableBody, _ := json.Marshal(dto.DisableTwoFARequest{Password: dto.Password{Password: "pass123"}})
	req = httptest.NewRequest(http.MethodPut, "/users/2fa/disable", bytes.NewBuffer(disableBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp = httptest.NewRecorder()
	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("disable failed: %d", resp.Code)
	}
}

func TestUsersRouter_2FA_Failures(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	svc := service.NewUserService(env.dep)
	u := createUser(t, env.dep, "2fafail", "2fafail@e.com", "pass123")
	token := addUserToken(t, env.dep, u.ID)

	setupResp, _ := svc.StartTwoFaSetup(context.Background(), u.ID)

	cases := []struct {
		name       string
		method     string
		path       string
		body       any
		wantStatus int
		setup      func()
	}{
		{"InvalidCode", http.MethodPost, "/users/2fa/confirm", dto.TwoFAConfirmRequest{SetupToken: setupResp.SetupToken, TwoFACode: "000000"}, http.StatusBadRequest, nil},
		{"SetupAlreadyEnabled", http.MethodPost, "/users/2fa/setup", nil, http.StatusBadRequest, func() {
			code, _ := totp.GenerateCode(setupResp.TwoFASecret, time.Now())
			confirmRes, _ := svc.ConfirmTwoFaSetup(context.Background(), u.ID, &dto.TwoFAConfirmRequest{SetupToken: setupResp.SetupToken, TwoFACode: code})
			token = confirmRes.Token
		}},
		{"WrongDisablePassword", http.MethodPut, "/users/2fa/disable", dto.DisableTwoFARequest{Password: dto.Password{Password: "wrong123"}}, http.StatusUnauthorized, func() {
			if token == "" {
				code, _ := totp.GenerateCode(setupResp.TwoFASecret, time.Now())
				confirmRes, _ := svc.ConfirmTwoFaSetup(context.Background(), u.ID, &dto.TwoFAConfirmRequest{SetupToken: setupResp.SetupToken, TwoFACode: code})
				token = confirmRes.Token
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			var body []byte
			if tc.body != nil {
				body, _ = json.Marshal(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+token)
			if tc.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			resp := httptest.NewRecorder()
			env.router.ServeHTTP(resp, req)
			if resp.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.Code)
			}
		})
	}
}

func TestUsersRouter_GoogleOAuth(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	req := httptest.NewRequest(http.MethodGet, "/users/google/login", nil)
	resp := httptest.NewRecorder()
	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", resp.Code)
	}
	if loc := resp.Header().Get("Location"); loc == "" {
		t.Error("expected location header")
	}

	origExchange := service.ExchangeCodeForTokens
	origFetch := service.FetchGoogleUserInfo
	defer func() {
		service.ExchangeCodeForTokens = origExchange
		service.FetchGoogleUserInfo = origFetch
	}()

	service.ExchangeCodeForTokens = func(_ *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
		return &idtoken.Payload{Subject: "g123"}, nil
	}
	service.FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
		return &dto.GoogleUserData{ID: "g123", Email: "test@google.com", Name: "Google User"}, nil
	}

	state, _ := jwt.SignOauthStateToken(env.dep)
	req = httptest.NewRequest(http.MethodGet, "/users/google/callback?code=valid&state="+state, nil)
	resp = httptest.NewRecorder()
	env.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", resp.Code)
	}

	redirectURL, _ := url.Parse(resp.Header().Get("Location"))
	token := redirectURL.Query().Get("token")
	if token == "" {
		t.Error("expected token in redirect")
	}
}

func TestUsersRouter_GoogleOAuth_Failures(t *testing.T) {
	env := setupUsersRouterTest(t, false)
	defer env.cleanup()

	cases := []struct {
		name       string
		path       string
		setup      func()
		wantStatus int
	}{
		{"MissingParams", "/users/google/callback", nil, http.StatusBadRequest},
		{"ExchangeError", "/users/google/callback?code=c&state=state", func() {
			origExchange := service.ExchangeCodeForTokens
			service.ExchangeCodeForTokens = func(_ *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
				return nil, errors.New("mock error")
			}
			t.Cleanup(func() { service.ExchangeCodeForTokens = origExchange })
		}, http.StatusFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			path := tc.path
			if strings.Contains(path, "state=state") {
				state, _ := jwt.SignOauthStateToken(env.dep)
				path = "/users/google/callback?code=c&state=" + state
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp := httptest.NewRecorder()
			env.router.ServeHTTP(resp, req)
			if resp.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.Code)
			}
			if tc.name == "ExchangeError" {
				loc := resp.Header().Get("Location")
				if !strings.Contains(loc, "error=") {
					t.Fatalf("expected error param in redirect: %s", loc)
				}
			}
		})
	}
}

func TestUsersRouter_Redis_LoginValidateLogout(t *testing.T) {
	env := setupUsersRouterTest(t, true)
	defer env.cleanup()

	createReq := dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "redisrouter"}, Email: "redisrouter@example.com"},
		Password: dto.Password{Password: "password123"},
	}
	createBody, _ := json.Marshal(createReq)
	createResp := httptest.NewRecorder()
	createHTTP := httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBuffer(createBody))
	createHTTP.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(createResp, createHTTP)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 on create, got %d. Body: %s", createResp.Code, createResp.Body.String())
	}

	loginReq := dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "redisrouter"},
		Password:   dto.Password{Password: "password123"},
	}
	loginBody, _ := json.Marshal(loginReq)
	loginResp := httptest.NewRecorder()
	loginHTTP := httptest.NewRequest(http.MethodPost, "/users/loginByIdentifier", bytes.NewBuffer(loginBody))
	loginHTTP.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(loginResp, loginHTTP)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d. Body: %s", loginResp.Code, loginResp.Body.String())
	}

	var loginUser dto.UserWithTokenResponse
	_ = json.Unmarshal(loginResp.Body.Bytes(), &loginUser)
	if loginUser.Token == "" || loginUser.ID == 0 {
		t.Fatalf("expected login to return token and id")
	}

	validateResp := httptest.NewRecorder()
	validateHTTP := httptest.NewRequest(http.MethodPost, "/users/validate", nil)
	validateHTTP.Header.Set("Authorization", "Bearer "+loginUser.Token)
	env.router.ServeHTTP(validateResp, validateHTTP)
	if validateResp.Code != http.StatusOK {
		t.Fatalf("expected 200 on validate, got %d. Body: %s", validateResp.Code, validateResp.Body.String())
	}

	logoutResp := httptest.NewRecorder()
	logoutHTTP := httptest.NewRequest(http.MethodDelete, "/users/logout", nil)
	logoutHTTP.Header.Set("Authorization", "Bearer "+loginUser.Token)
	env.router.ServeHTTP(logoutResp, logoutHTTP)
	if logoutResp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on logout, got %d. Body: %s", logoutResp.Code, logoutResp.Body.String())
	}

	validateAfterResp := httptest.NewRecorder()
	validateAfterHTTP := httptest.NewRequest(http.MethodPost, "/users/validate", nil)
	validateAfterHTTP.Header.Set("Authorization", "Bearer "+loginUser.Token)
	env.router.ServeHTTP(validateAfterResp, validateAfterHTTP)
	if validateAfterResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on validate after logout, got %d. Body: %s", validateAfterResp.Code, validateAfterResp.Body.String())
	}

	time.Sleep(200 * time.Millisecond)
	score, err := env.dep.Redis.ZScore(context.Background(), "heartbeat:", strconv.FormatUint(uint64(loginUser.ID), 10)).Result()
	if err != nil {
		t.Fatalf("expected heartbeat entry, got error: %v", err)
	}
	if int64(score) < time.Now().Unix()-10 {
		t.Fatalf("expected recent heartbeat score, got %v", score)
	}
}
