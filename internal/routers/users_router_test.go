package routers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/auth/credentials/idtoken"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/service"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

func setupUsersRouterTestUnique(t *testing.T) (*gin.Engine, func()) {
	gin.SetMode(gin.TestMode)
	
	// Mock Logger
	util.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Mock Config
	prevCfg := config.Cfg
	config.Cfg = &config.Config{
		JwtSecret:             "test-secret",
		UserTokenExpiry:       3600,
		TwoFaTokenExpiry:      3600,
		OauthStateTokenExpiry: 3600,
		GoogleClientId:        "test-client",
		GoogleRedirectUri:     "http://localhost/cb",
		FrontendUrl:           "http://localhost:3000",
	}
	dto.InitValidator()

	// Mock DB
	var err error
	// Use unique DB name for each test run to avoid lock issues
	// Sanitize test name
	dbName := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_busy_timeout=5000"
	
	model.DB, err = gorm.Open(sqlite.Open(dbName), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	
	// Explicitly enable foreign keys
	model.DB.Exec("PRAGMA foreign_keys = ON")

	err = model.DB.AutoMigrate(&model.User{}, &model.Friend{}, &model.Token{}, &model.HeartBeat{})
	if err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	router := gin.New()
	UsersRouter(router.Group("/users"))

	// Set MaxOpenConns to 1 to avoid locking issues in tests with SQLite
	if model.DB != nil {
		sqlDB, _ := model.DB.DB()
		if sqlDB != nil {
			sqlDB.SetMaxOpenConns(1)
		}
	}

	return router, func() {
		config.Cfg = prevCfg
		if model.DB != nil {
			sqlDB, _ := model.DB.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
			model.DB = nil
		}
	}
}

func TestUsersRouter_CreateUser(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	reqBody := dto.CreateUserRequest{
		User: dto.User{UserName: dto.UserName{Username: "newuser"}, Email: "new@example.com"},
		Password: dto.Password{Password: "password123"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var user dto.UserWithoutTokenResponse
	json.Unmarshal(resp.Body.Bytes(), &user)
	if user.Username != "newuser" {
		t.Errorf("expected username newuser, got %s", user.Username)
	}
}

func TestUsersRouter_LoginUser(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	// Create user
	createReq := dto.CreateUserRequest{
		User: dto.User{UserName: dto.UserName{Username: "loginuser"}, Email: "login@example.com"},
		Password: dto.Password{Password: "password123"},
	}
	createBody, _ := json.Marshal(createReq)
	
	cReq := httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBuffer(createBody))
	cReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), cReq)

	// Login
	loginReq := dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "loginuser"},
		Password:   dto.Password{Password: "password123"},
	}
	body, _ := json.Marshal(loginReq)

	req := httptest.NewRequest(http.MethodPost, "/users/loginByIdentifier", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var res service.LoginResult
	json.Unmarshal(resp.Body.Bytes(), &res)
	if res.User == nil || res.User.Token == "" {
		t.Errorf("expected token in response. Body: %s", resp.Body.String())
	}
}

func TestUsersRouter_GetProfile(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	user := model.User{
		Username: "profileuser", 
		Email: "profile@example.com",
	}
	model.DB.Create(&user)

	tokenStr, _ := jwt.SignUserToken(user.ID)
	model.DB.Create(&model.Token{UserID: user.ID, Token: tokenStr})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var res dto.UserWithoutTokenResponse
	json.Unmarshal(resp.Body.Bytes(), &res)
	if res.Username != "profileuser" {
		t.Errorf("expected username profileuser, got %s", res.Username)
	}
}

func TestUsersRouter_Unathorized(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.Code)
	}
}

func TestUsersRouter_UpdateUserProfile(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	user := model.User{Username: "u", Email: "u@e.com"}
	model.DB.Create(&user)
	tokenStr, _ := jwt.SignUserToken(user.ID)
	model.DB.Create(&model.Token{UserID: user.ID, Token: tokenStr})

	newAvatar := "http://pic.com/1.png"
	reqBody := dto.UpdateUserRequest{
		User: dto.User{UserName: dto.UserName{Username: "newname"}, Email: "new@e.com", Avatar: &newAvatar},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var res dto.UserWithoutTokenResponse
	json.Unmarshal(resp.Body.Bytes(), &res)
	if res.Username != "newname" {
		t.Errorf("expected new username, got %s", res.Username)
	}
}

func TestUsersRouter_UpdateUserPassword(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	svc := service.NewUserService(model.DB)
	userResp, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User: dto.User{UserName: dto.UserName{Username: "pw"}, Email: "pw@e.com"},
		Password: dto.Password{Password: "oldpass"},
	})
	tokenStr, _ := jwt.SignUserToken(userResp.ID)
	model.DB.Create(&model.Token{UserID: userResp.ID, Token: tokenStr})

	reqBody := dto.UpdateUserPasswordRequest{
		OldPassword: dto.OldPassword{OldPassword: "oldpass"},
		NewPassword: dto.NewPassword{NewPassword: "newpass"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPut, "/users/password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}
}

func TestUsersRouter_DeleteUser(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	user := model.User{Username: "del", Email: "del@e.com"}
	model.DB.Create(&user)
	tokenStr, _ := jwt.SignUserToken(user.ID)
	model.DB.Create(&model.Token{UserID: user.ID, Token: tokenStr})

	// Let DB settle
	time.Sleep(500 * time.Millisecond)

	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.Code)
	}
}

func TestUsersRouter_GetUsersWithLimitedInfo(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	user := model.User{Username: "list", Email: "list@e.com"}
	model.DB.Create(&user)
	tokenStr, _ := jwt.SignUserToken(user.ID)
	model.DB.Create(&model.Token{UserID: user.ID, Token: tokenStr})

	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestUsersRouter_ValidateUser(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	user := model.User{Username: "val", Email: "val@e.com"}
	model.DB.Create(&user)
	tokenStr, _ := jwt.SignUserToken(user.ID)
	model.DB.Create(&model.Token{UserID: user.ID, Token: tokenStr})

	req := httptest.NewRequest(http.MethodPost, "/users/validate", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	
	var res dto.UserValidationResponse
	json.Unmarshal(resp.Body.Bytes(), &res)
	if res.UserID != user.ID {
		t.Errorf("expected userID %d, got %d", user.ID, res.UserID)
	}
}

func TestUsersRouter_Friends(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	svc := service.NewUserService(model.DB)
	u1, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User: dto.User{UserName: dto.UserName{Username: "f1"}, Email: "f1@e.com"},
		Password: dto.Password{Password: "p"},
	})
	u2, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User: dto.User{UserName: dto.UserName{Username: "f2"}, Email: "f2@e.com"},
		Password: dto.Password{Password: "p"},
	})

	tokenStr, _ := jwt.SignUserToken(u1.ID)
	model.DB.Create(&model.Token{UserID: u1.ID, Token: tokenStr})

	// Add Friend
	reqBody := dto.AddNewFriendRequest{UserID: u2.ID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/users/friends", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	// Get Friends
	req = httptest.NewRequest(http.MethodGet, "/users/friends", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	var friends []dto.FriendResponse
	json.Unmarshal(resp.Body.Bytes(), &friends)
	if len(friends) != 1 || friends[0].ID != u2.ID {
		t.Error("expected friend f2")
	}
}

func TestUsersRouter_2FA(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	svc := service.NewUserService(model.DB)
	user, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User: dto.User{UserName: dto.UserName{Username: "2fa"}, Email: "2fa@e.com"},
		Password: dto.Password{Password: "pass"},
	})
	tokenStr, _ := jwt.SignUserToken(user.ID)
	model.DB.Create(&model.Token{UserID: user.ID, Token: tokenStr})

	// 1. Setup 2FA
	req := httptest.NewRequest(http.MethodPost, "/users/2fa/setup", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("setup failed: %d", resp.Code)
	}
	var setupRes dto.TwoFASetupResponse
	json.Unmarshal(resp.Body.Bytes(), &setupRes)

	// Let DB settle
	time.Sleep(200 * time.Millisecond)

	// 2. Confirm 2FA
	code, _ := totp.GenerateCode(setupRes.TwoFASecret, time.Now())
	confirmBody, _ := json.Marshal(dto.TwoFAConfirmRequest{
		SetupToken: setupRes.SetupToken,
		TwoFACode:  code,
	})
	req = httptest.NewRequest(http.MethodPost, "/users/2fa/confirm", bytes.NewBuffer(confirmBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("confirm failed: %d", resp.Code)
	}

	// 3. Login challenge
	// Need pending session token
	sessionToken, _ := jwt.SignTwoFAToken(user.ID)
	code, _ = totp.GenerateCode(setupRes.TwoFASecret, time.Now())
	challengeBody, _ := json.Marshal(dto.TwoFAChallengeRequest{
		SessionToken: sessionToken,
		TwoFACode:    code,
	})
	req = httptest.NewRequest(http.MethodPost, "/users/2fa", bytes.NewBuffer(challengeBody))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("challenge failed: %d body: %s", resp.Code, resp.Body.String())
	}

	// 4. Disable 2FA
	// We need a valid user token again (new one from confirm or challenge, or reuse old if valid)
	// But `issueNewTokenForUser` revokes old tokens if true passed. Confirm passed true.
	// So tokenStr is invalid. We need the one from confirm response.
	var userRes dto.UserWithTokenResponse
	json.Unmarshal(resp.Body.Bytes(), &userRes)
	tokenStr = userRes.Token

	// Let DB settle
	time.Sleep(200 * time.Millisecond)

	disableBody, _ := json.Marshal(dto.DisableTwoFARequest{
		Password: dto.Password{Password: "pass"},
	})
	req = httptest.NewRequest(http.MethodPut, "/users/2fa/disable", bytes.NewBuffer(disableBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("disable failed: %d", resp.Code)
	}
}

func TestUsersRouter_GoogleOAuth(t *testing.T) {
	router, cleanup := setupUsersRouterTestUnique(t)
	defer cleanup()

	// 1. Google Login (Redirect)
	req := httptest.NewRequest(http.MethodGet, "/users/google/login", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", resp.Code)
	}
	// Verify location header format roughly
	loc := resp.Header().Get("Location")
	if loc == "" {
		t.Error("expected location header")
	}

	// 2. Google Callback
	// Mock service vars
	origExchange := service.ExchangeCodeForTokens
	origFetch := service.FetchGoogleUserInfo
	defer func() {
		service.ExchangeCodeForTokens = origExchange
		service.FetchGoogleUserInfo = origFetch
	}()

	service.ExchangeCodeForTokens = func(ctx context.Context, code string) (*idtoken.Payload, error) {
		return &idtoken.Payload{Subject: "g123"}, nil
	}
	service.FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
		return &dto.GoogleUserData{
			ID:    "g123",
			Email: "test@google.com",
			Name:  "Google User",
		}, nil
	}

	// Generate state
	state, _ := jwt.SignOauthStateToken()

	req = httptest.NewRequest(http.MethodGet, "/users/google/callback?code=valid&state="+state, nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", resp.Code)
	}
	
	redirectURL, _ := url.Parse(resp.Header().Get("Location"))
	token := redirectURL.Query().Get("token")
	if token == "" {
		t.Error("expected token in redirect")
	}
}
