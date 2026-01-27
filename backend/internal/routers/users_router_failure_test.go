package routers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

func setupUsersRouterTestFailure(t *testing.T) (*gin.Engine, func()) {
	gin.SetMode(gin.TestMode)

	util.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

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

	dbName := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared&_busy_timeout=5000&_foreign_keys=on"
	var err error
	model.DB, err = gorm.Open(sqlite.Open(dbName), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	model.DB.Exec("PRAGMA foreign_keys = ON")

	err = model.DB.AutoMigrate(&model.User{}, &model.Friend{}, &model.Token{}, &model.HeartBeat{})
	if err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	router := gin.New()
	UsersRouter(router.Group("/users"))

	// Set MaxOpenConns to 1 to avoid locking issues
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
				_ = sqlDB.Close()
			}
			model.DB = nil
		}
	}
}

func TestUsersRouter_CreateUser_Failures(t *testing.T) {
	router, cleanup := setupUsersRouterTestFailure(t)
	defer cleanup()

	// 1. Invalid Body
	reqBody := `{"username": "u"}` // Missing email, password, invalid username length
	req := httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", resp.Code)
	}

	// 2. Duplicate User
	// Create first
	validReq := dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "dupuser"}, Email: "dup@e.com"},
		Password: dto.Password{Password: "pass"},
	}
	body, _ := json.Marshal(validReq)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBuffer(body)))

	// Try duplicate
	req = httptest.NewRequest(http.MethodPost, "/users/", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d", resp.Code)
	}
}

func TestUsersRouter_LoginUser_Failures(t *testing.T) {
	router, cleanup := setupUsersRouterTestFailure(t)
	defer cleanup()

	// 1. Invalid Body
	req := httptest.NewRequest(http.MethodPost, "/users/loginByIdentifier", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", resp.Code)
	}

	// 2. User Not Found
	loginReq := dto.LoginUserRequest{
		Identifier: dto.Identifier{Identifier: "missing"},
		Password:   dto.Password{Password: "pass"},
	}
	body, _ := json.Marshal(loginReq)
	req = httptest.NewRequest(http.MethodPost, "/users/loginByIdentifier", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing user, got %d", resp.Code)
	}

	// 3. Invalid Credentials
	// Create user
	svc := service.NewUserService(model.DB, nil)
	_, _ = svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "loginfail"}, Email: "fail@e.com"},
		Password: dto.Password{Password: "correct"},
	})

	loginReq.Identifier.Identifier = "loginfail"
	loginReq.Password.Password = "wrong"
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest(http.MethodPost, "/users/loginByIdentifier", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", resp.Code)
	}
}

func TestUsersRouter_UpdateUser_Failures(t *testing.T) {
	router, cleanup := setupUsersRouterTestFailure(t)
	defer cleanup()

	svc := service.NewUserService(model.DB, nil)
	u, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "u1"}, Email: "u1@e.com"},
		Password: dto.Password{Password: "pass"},
	})
	token, _ := jwt.SignUserToken(u.ID)
	model.DB.Create(&model.Token{UserID: u.ID, Token: token})

	// 1. Update Profile Duplicate
	// Create another user
	_, _ = svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "u2"}, Email: "u2@e.com"},
		Password: dto.Password{Password: "pass"},
	})

	updateReq := dto.UpdateUserRequest{
		User: dto.User{UserName: dto.UserName{Username: "update_u2"}, Email: "u2@e.com"},
	}
	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate profile update, got %d", resp.Code)
	}

	// 2. Update Password Wrong Old
	pwReq := dto.UpdateUserPasswordRequest{
		OldPassword: dto.OldPassword{OldPassword: "wrong"},
		NewPassword: dto.NewPassword{NewPassword: "newpass"},
	}
	body, _ = json.Marshal(pwReq)
	req = httptest.NewRequest(http.MethodPut, "/users/password", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong old password, got %d", resp.Code)
	}
}

func TestUsersRouter_Friends_Failures(t *testing.T) {
	router, cleanup := setupUsersRouterTestFailure(t)
	defer cleanup()

	svc := service.NewUserService(model.DB, nil)
	u1, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "f1"}, Email: "f1@e.com"},
		Password: dto.Password{Password: "pass"},
	})
	u2, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "f2"}, Email: "f2@e.com"},
		Password: dto.Password{Password: "pass"},
	})
	token, _ := jwt.SignUserToken(u1.ID)
	model.DB.Create(&model.Token{UserID: u1.ID, Token: token})

	// 1. Add Self
	reqBody := dto.AddNewFriendRequest{UserID: u1.ID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/users/friends", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for adding self, got %d", resp.Code)
	}

	// 2. Add Non-existent
	reqBody = dto.AddNewFriendRequest{UserID: 999}
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/users/friends", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing friend, got %d", resp.Code)
	}

	// 3. Duplicate Friend
	_ = svc.AddNewFriend(context.Background(), u1.ID, &dto.AddNewFriendRequest{UserID: u2.ID})

	// Let DB settle
	time.Sleep(200 * time.Millisecond)

	reqBody = dto.AddNewFriendRequest{UserID: u2.ID}
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/users/friends", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate friend, got %d", resp.Code)
	}
}

func TestUsersRouter_2FA_Failures(t *testing.T) {
	router, cleanup := setupUsersRouterTestFailure(t)
	defer cleanup()

	svc := service.NewUserService(model.DB, nil)
	u, _ := svc.CreateUser(context.Background(), &dto.CreateUserRequest{
		User:     dto.User{UserName: dto.UserName{Username: "2fafail"}, Email: "2fafail@e.com"},
		Password: dto.Password{Password: "pass"},
	})
	token, _ := jwt.SignUserToken(u.ID)
	model.DB.Create(&model.Token{UserID: u.ID, Token: token})

	// 1. Confirm with invalid code
	setupResp, _ := svc.StartTwoFaSetup(context.Background(), u.ID)
	confirmReq := dto.TwoFAConfirmRequest{
		SetupToken: setupResp.SetupToken,
		TwoFACode:  "000000",
	}
	body, _ := json.Marshal(confirmReq)
	req := httptest.NewRequest(http.MethodPost, "/users/2fa/confirm", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid code, got %d", resp.Code)
	}

	// 2. Start Setup when Already Enabled
	// Enable it correctly first
	code, _ := totp.GenerateCode(setupResp.TwoFASecret, time.Now())
	confirmRes, _ := svc.ConfirmTwoFaSetup(context.Background(), u.ID, &dto.TwoFAConfirmRequest{SetupToken: setupResp.SetupToken, TwoFACode: code})

	// Update token as confirming 2FA issues a new one
	token = confirmRes.Token

	req = httptest.NewRequest(http.MethodPost, "/users/2fa/setup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for setup when already enabled, got %d", resp.Code)
	}

	// 3. Disable with Wrong Password
	disableReq := dto.DisableTwoFARequest{Password: dto.Password{Password: "wrong"}}
	body, _ = json.Marshal(disableReq)
	req = httptest.NewRequest(http.MethodPut, "/users/2fa/disable", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password disable, got %d", resp.Code)
	}
}

func TestUsersRouter_GoogleOAuth_Failures(t *testing.T) {
	router, cleanup := setupUsersRouterTestFailure(t)
	defer cleanup()

	// 1. Missing Params
	req := httptest.NewRequest(http.MethodGet, "/users/google/callback", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest { // 400 from handler check
		t.Errorf("expected 400 for missing params, got %d", resp.Code)
	}

	// 2. Invalid State/Code (Service Fail)
	// We mocked service vars in other test file, but here they are originals unless we mock them again.
	// Since setupUsersRouterTestFailure is separate, vars are global in `service` package.
	// We should mock them to RETURN ERROR.

	origExchange := service.ExchangeCodeForTokens
	defer func() { service.ExchangeCodeForTokens = origExchange }()
	service.ExchangeCodeForTokens = func(ctx context.Context, code string) (*idtoken.Payload, error) {
		return nil, errors.New("mock error")
	}

	state, _ := jwt.SignOauthStateToken()
	req = httptest.NewRequest(http.MethodGet, "/users/google/callback?code=c&state="+state, nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusFound { // Redirect to error page
		t.Errorf("expected 302 redirect to error, got %d", resp.Code)
	}
	loc := resp.Header().Get("Location")
	if !strings.Contains(loc, "error=") {
		t.Errorf("expected error param in redirect: %s", loc)
	}
}
