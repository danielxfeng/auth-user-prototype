package routers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"

	"github.com/gin-gonic/gin"
	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/routers"
	"github.com/paularynty/transcendence/auth-service-go/internal/service"
	"github.com/paularynty/transcendence/auth-service-go/internal/testutil"
)

func testRouterFactory(t *testing.T, testCfg *config.Config, setDBDown bool) *gin.Engine {
	t.Helper()

	dto.InitValidator()

	testLogger := testutil.NewTestLogger()

	testCfg.DbAddress = testutil.GetSafeTestDBName(testCfg.DbAddress, t.Name())

	myDB, err := db.GetDB(testCfg.DbAddress, testLogger)
	if err != nil {
		t.Fatalf("failed to init the test db, err: %v", err)
	}
	db.ResetDB(myDB, testLogger)

	var redisClient *redis.Client

	if testCfg.IsRedisEnabled {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis, err: %v", err)
		}
		t.Cleanup(func() {
			mr.Close()
		})

		testCfg.RedisURL = "redis://" + mr.Addr()
		redisClient, err = db.GetRedis(testCfg.RedisURL, testCfg, testLogger)
		if err != nil {
			t.Fatalf("failed to init the test redis, err: %v", err)
		}
		t.Cleanup(func() {
			db.CloseRedis(redisClient, testLogger)
		})
	}

	dep := testutil.NewTestDependency(testCfg, myDB, redisClient, testLogger)

	userService, err := service.NewUserService(dep)

	if err != nil {
		t.Fatalf("faled to create user service")
	}

	if setDBDown {
		userService.Dep.DB = nil
	}

	r := routers.SetupRouter(dep)
	routers.UsersRouter(r.Group("/"), userService)

	return r
}

func toJSON(t *testing.T, v any) *strings.Reader {
	t.Helper()

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("failed to serialize the obj %v, err: %v", v, err)
	}
	return strings.NewReader(string(b))
}

var testUsername1 = "test1"
var testEmail1 = "test1@test.com"
var testUsername2 = "test2"
var testEmail2 = "test2@test.com"
var testPwd = "Password.777"
var testNewPassword = "Password.888"
var testAvatar = "https://example.com/a.png"
var loginUsername1 = "loginuser1"
var loginEmail1 = "loginuser1@test.com"

var mockRegisterRequest = map[string]string{
	"username": testUsername1,
	"email":    testEmail1,
	"password": testPwd,
}

var mockRegisterRequest2 = map[string]string{
	"username": testUsername2,
	"email":    testEmail2,
	"password": testPwd,
}

var mockUpdateUserPasswordRequest = map[string]string{
	"oldPassword": testPwd,
	"newPassword": testNewPassword,
}

var mockLoginUserByUsernameRequest = map[string]string{
	"identifier": testUsername1,
	"password":   testPwd,
}

var mockLoginUserByEmailRequest = map[string]string{
	"identifier": testEmail1,
	"password":   testPwd,
}

var mockUpdateUserRequest = map[string]string{
	"username": testUsername1,
	"email":    testEmail1,
	"avatar":   testAvatar,
}

func TestCreateUserEndpoint(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(t *testing.T, r *gin.Engine)
		body  any
		want  int
	}{
		{
			name: "happy",
			body: mockRegisterRequest,
			want: 201,
		},
		{
			name: "duplicate username",
			setup: func(t *testing.T, r *gin.Engine) {
				userReq := map[string]string{
					"username": "dupusername",
					"email":    "dupusername@test.com",
					"password": testPwd,
				}
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/", toJSON(t, userReq))
				r.ServeHTTP(w, req)
				if w.Code != 201 {
					t.Fatalf("setup register failed, got %d", w.Code)
				}
			},
			body: map[string]string{
				"username": "dupusername",
				"email":    "other_dupusername@test.com",
				"password": testPwd,
			},
			want: 409,
		},
		{
			name: "duplicate email",
			setup: func(t *testing.T, r *gin.Engine) {
				userReq := map[string]string{
					"username": "dupemail",
					"email":    "dupemail@test.com",
					"password": testPwd,
				}
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/", toJSON(t, userReq))
				r.ServeHTTP(w, req)
				if w.Code != 201 {
					t.Fatalf("setup register failed, got %d", w.Code)
				}
			},
			body: map[string]string{
				"username": "other_dupemail",
				"email":    "dupemail@test.com",
				"password": testPwd,
			},
			want: 409,
		},
		{
			name: "invalid email",
			body: map[string]string{
				"username": testUsername1,
				"email":    "not-an-email",
				"password": testPwd,
			},
			want: 400,
		},
		{name: "missing body", body: nil, want: 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCfg := testutil.NewTestConfig()
			testCfg.RedisURL = "redis"
			testCfg.IsRedisEnabled = true
			testCfg.RateLimiterRequestLimit = 1000
			r := testRouterFactory(t, testCfg, false)

			body := tc.body
			if tc.setup != nil {
				tc.setup(t, r)
			}

			w := httptest.NewRecorder()
			var bodyReader io.Reader
			if body != nil {
				bodyReader = toJSON(t, body)
			} else {
				bodyReader = strings.NewReader("")
			}
			req, _ := http.NewRequest("POST", "/", bodyReader)
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected: %d, got %d", tc.want, w.Code)
			}
		})
	}
}

func TestLoginEndpoint(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RateLimiterRequestLimit = 1000
	r := testRouterFactory(t, testCfg, false)

	// Register a user once for login tests.
	registerReq := map[string]string{
		"username": loginUsername1,
		"email":    loginEmail1,
		"password": testPwd,
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", toJSON(t, registerReq))
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("setup register failed, got %d", w.Code)
	}

	testCases := []struct {
		name string
		body any
		want int
	}{
		{
			name: "happy username",
			body: map[string]string{
				"identifier": loginUsername1,
				"password":   testPwd,
			},
			want: 200,
		},
		{
			name: "happy email",
			body: map[string]string{
				"identifier": loginEmail1,
				"password":   testPwd,
			},
			want: 200,
		},
		{
			name: "wrong password",
			body: map[string]string{
				"identifier": loginEmail1,
				"password":   "WrongPassword.123",
			},
			want: 401,
		},
		{
			name: "wrong identifier",
			body: map[string]string{
				"identifier": "unknown@test.com",
				"password":   testPwd,
			},
			want: 401,
		},
		{
			name: "missing body",
			body: nil,
			want: 400,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			var bodyReader io.Reader
			if tc.body != nil {
				bodyReader = toJSON(t, tc.body)
			} else {
				bodyReader = strings.NewReader("")
			}
			req, _ := http.NewRequest("POST", "/loginByIdentifier", bodyReader)
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected: %d, got %d", tc.want, w.Code)
			}
		})
	}
}

func TestUpdatePasswordEndpoint(t *testing.T) {
	testCases := []struct {
		name string
		body any
		want int
	}{
		{name: "happy", body: mockUpdateUserPasswordRequest, want: 200},
		{
			name: "wrong old password",
			body: map[string]string{
				"oldPassword": "WrongPassword.123",
				"newPassword": testNewPassword,
			},
			want: 401,
		},
		{name: "missing body", body: nil, want: 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCfg := testutil.NewTestConfig()
			testCfg.RateLimiterRequestLimit = 1000
			r := testRouterFactory(t, testCfg, false)

			// setup user + login
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest))
			r.ServeHTTP(w, req)
			if w.Code != 201 {
				t.Fatalf("setup register failed, got %d", w.Code)
			}

			w = httptest.NewRecorder()
			req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
			r.ServeHTTP(w, req)
			if w.Code != 200 {
				t.Fatalf("setup login failed, got %d", w.Code)
			}

			var login dto.UserWithTokenResponse
			if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
				t.Fatalf("failed to unmarshal login response: %v", err)
			}

			var secondToken string
			if tc.name == "happy" {
				w = httptest.NewRecorder()
				req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
				r.ServeHTTP(w, req)
				if w.Code != 200 {
					t.Fatalf("setup second login failed, got %d", w.Code)
				}
				var login2 dto.UserWithTokenResponse
				if err := json.Unmarshal(w.Body.Bytes(), &login2); err != nil {
					t.Fatalf("failed to unmarshal second login response: %v", err)
				}
				secondToken = login2.Token
			}

			w = httptest.NewRecorder()
			var bodyReader io.Reader
			if tc.body != nil {
				bodyReader = toJSON(t, tc.body)
			} else {
				bodyReader = strings.NewReader("")
			}
			req, _ = http.NewRequest("PUT", "/password", bodyReader)
			req.Header.Add("Authorization", "Bearer "+login.Token)
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected: %d, got %d", tc.want, w.Code)
			}

			if tc.name == "happy" {
				// Old token should be invalid after password update
				w = httptest.NewRecorder()
				req, _ = http.NewRequest("POST", "/validate", nil)
				req.Header.Add("Authorization", "Bearer "+login.Token)
				r.ServeHTTP(w, req)
				if w.Code != 401 {
					t.Fatalf("expected old token to be invalid after password update, got %d", w.Code)
				}

				if secondToken == "" {
					t.Fatalf("expected second token to be set")
				}
				w = httptest.NewRecorder()
				req, _ = http.NewRequest("POST", "/validate", nil)
				req.Header.Add("Authorization", "Bearer "+secondToken)
				r.ServeHTTP(w, req)
				if w.Code != 401 {
					t.Fatalf("expected second token to be invalid after password update, got %d", w.Code)
				}
			}
		})
	}
}

func TestUpdateProfileEndpoint(t *testing.T) {
	createUser := func(t *testing.T, r *gin.Engine, body any) dto.UserWithoutTokenResponse {
		t.Helper()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/", toJSON(t, body))
		r.ServeHTTP(w, req)
		if w.Code != 201 {
			t.Fatalf("setup register failed, got %d", w.Code)
		}
		var user dto.UserWithoutTokenResponse
		if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
			t.Fatalf("failed to unmarshal register response: %v", err)
		}
		return user
	}

	loginUser := func(t *testing.T, r *gin.Engine, body any) dto.UserWithTokenResponse {
		t.Helper()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/loginByIdentifier", toJSON(t, body))
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("setup login failed, got %d", w.Code)
		}
		var user dto.UserWithTokenResponse
		if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
			t.Fatalf("failed to unmarshal login response: %v", err)
		}
		return user
	}

	testCases := []struct {
		name  string
		body  any
		want  int
		check func(t *testing.T, body []byte)
	}{
		{
			name: "happy",
			body: mockUpdateUserRequest,
			want: 200,
		},
		{
			name: "avatar null",
			body: map[string]any{
				"username": testUsername1,
				"email":    testEmail1,
				"avatar":   nil,
			},
			want: 200,
			check: func(t *testing.T, body []byte) {
				var resp dto.UserWithoutTokenResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Avatar != nil {
					t.Fatalf("expected avatar to be nil, got %v", *resp.Avatar)
				}
			},
		},
		{
			name: "avatar empty",
			body: map[string]any{
				"username": testUsername1,
				"email":    testEmail1,
				"avatar":   "",
			},
			want: 400,
			check: func(t *testing.T, body []byte) {
				var resp struct {
					Error []string `json:"error"`
				}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal error response: %v", err)
				}
				if len(resp.Error) == 0 {
					t.Fatalf("expected validation errors, got none")
				}
				found := false
				for _, msg := range resp.Error {
					if strings.Contains(msg, "Avatar") {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected avatar validation error, got %v", resp.Error)
				}
			},
		},
		{
			name: "duplicate",
			body: map[string]string{
				"username": testUsername2,
				"email":    testEmail2,
				"avatar":   testAvatar,
			},
			want: 409,
		},
		{name: "missing body", body: nil, want: 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCfg := testutil.NewTestConfig()
			testCfg.RateLimiterRequestLimit = 1000
			r := testRouterFactory(t, testCfg, false)

			createUser(t, r, mockRegisterRequest)
			createUser(t, r, mockRegisterRequest2)
			login := loginUser(t, r, mockLoginUserByEmailRequest)

			w := httptest.NewRecorder()
			var bodyReader io.Reader
			if tc.body != nil {
				bodyReader = toJSON(t, tc.body)
			} else {
				bodyReader = strings.NewReader("")
			}
			req, _ := http.NewRequest("PUT", "/me", bodyReader)
			req.Header.Add("Authorization", "Bearer "+login.Token)
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected: %d, got %d", tc.want, w.Code)
			}
			if tc.check != nil {
				tc.check(t, w.Body.Bytes())
			}
		})
	}
}

func TestFriendsEndpoints(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RedisURL = "redis"
	testCfg.IsRedisEnabled = true
	testCfg.RateLimiterRequestLimit = 1000

	testCases := []struct {
		name string
		run  func(t *testing.T, r *gin.Engine, token string, user1 dto.UserWithoutTokenResponse, user2 dto.UserWithoutTokenResponse)
		want int
	}{
		{
			name: "list empty",
			run: func(t *testing.T, r *gin.Engine, token string, _ dto.UserWithoutTokenResponse, _ dto.UserWithoutTokenResponse) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("GET", "/friends", nil)
				req.Header.Add("Authorization", "Bearer "+token)
				r.ServeHTTP(w, req)
				if w.Code != 200 {
					t.Fatalf("expected: 200, got %d", w.Code)
				}
			},
			want: 200,
		},
		{
			name: "add friend",
			run: func(t *testing.T, r *gin.Engine, token string, _ dto.UserWithoutTokenResponse, user2 dto.UserWithoutTokenResponse) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/friends", toJSON(t, map[string]uint{"userId": user2.ID}))
				req.Header.Add("Authorization", "Bearer "+token)
				r.ServeHTTP(w, req)
				if w.Code != 201 {
					t.Fatalf("expected: 201, got %d", w.Code)
				}
			},
			want: 201,
		},
		{
			name: "add friend duplicate",
			run: func(t *testing.T, r *gin.Engine, token string, _ dto.UserWithoutTokenResponse, user2 dto.UserWithoutTokenResponse) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/friends", toJSON(t, map[string]uint{"userId": user2.ID}))
				req.Header.Add("Authorization", "Bearer "+token)
				r.ServeHTTP(w, req)
				if w.Code != 201 {
					t.Fatalf("setup add friend failed, got %d", w.Code)
				}

				w = httptest.NewRecorder()
				req, _ = http.NewRequest("POST", "/friends", toJSON(t, map[string]uint{"userId": user2.ID}))
				req.Header.Add("Authorization", "Bearer "+token)
				r.ServeHTTP(w, req)
				if w.Code != 409 {
					t.Fatalf("expected: 409, got %d", w.Code)
				}
			},
			want: 409,
		},
		{
			name: "add friend self",
			run: func(t *testing.T, r *gin.Engine, token string, user1 dto.UserWithoutTokenResponse, _ dto.UserWithoutTokenResponse) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/friends", toJSON(t, map[string]uint{"userId": user1.ID}))
				req.Header.Add("Authorization", "Bearer "+token)
				r.ServeHTTP(w, req)
				if w.Code != 400 {
					t.Fatalf("expected: 400, got %d", w.Code)
				}
			},
			want: 400,
		},
		{
			name: "add friend not found",
			run: func(t *testing.T, r *gin.Engine, token string, _ dto.UserWithoutTokenResponse, _ dto.UserWithoutTokenResponse) {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/friends", toJSON(t, map[string]uint{"userId": 999999}))
				req.Header.Add("Authorization", "Bearer "+token)
				r.ServeHTTP(w, req)
				if w.Code != 404 {
					t.Fatalf("expected: 404, got %d", w.Code)
				}
			},
			want: 404,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := testRouterFactory(t, testCfg, false)

			// setup users + login
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest))
			r.ServeHTTP(w, req)
			if w.Code != 201 {
				t.Fatalf("setup register user1 failed, got %d", w.Code)
			}
			var user1 dto.UserWithoutTokenResponse
			if err := json.Unmarshal(w.Body.Bytes(), &user1); err != nil {
				t.Fatalf("failed to unmarshal user1 response: %v", err)
			}

			w = httptest.NewRecorder()
			req, _ = http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest2))
			r.ServeHTTP(w, req)
			if w.Code != 201 {
				t.Fatalf("setup register user2 failed, got %d", w.Code)
			}

			var user2 dto.UserWithoutTokenResponse
			if err := json.Unmarshal(w.Body.Bytes(), &user2); err != nil {
				t.Fatalf("failed to unmarshal user2 response: %v", err)
			}

			w = httptest.NewRecorder()
			req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
			r.ServeHTTP(w, req)
			if w.Code != 200 {
				t.Fatalf("setup login failed, got %d", w.Code)
			}
			var login dto.UserWithTokenResponse
			if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
				t.Fatalf("failed to unmarshal login response: %v", err)
			}

			tc.run(t, r, login.Token, user1, user2)
		})
	}
}

func TestValidateEndpoint(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RateLimiterRequestLimit = 1000
	r := testRouterFactory(t, testCfg, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest))
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("setup register failed, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("setup login failed, got %d", w.Code)
	}
	var login dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/validate", nil)
	req.Header.Add("Authorization", "Bearer "+login.Token)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected: 200, got %d", w.Code)
	}
}

func TestListUsersEndpoint(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RateLimiterRequestLimit = 1000
	r := testRouterFactory(t, testCfg, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest))
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("setup register user1 failed, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest2))
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("setup register user2 failed, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("setup login failed, got %d", w.Code)
	}
	var login dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/", nil)
	req.Header.Add("Authorization", "Bearer "+login.Token)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected: 200, got %d", w.Code)
	}
}

func TestGoogleOAuthEndpoints(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RateLimiterRequestLimit = 1000
	r := testRouterFactory(t, testCfg, false)

	t.Run("login redirect", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/google/login", nil)
		r.ServeHTTP(w, req)
		if w.Code != 302 {
			t.Fatalf("expected: 302, got %d", w.Code)
		}
		if w.Header().Get("Location") == "" {
			t.Fatalf("expected redirect location header")
		}
	})

	t.Run("callback missing code", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/google/callback?state=abc", nil)
		r.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("expected: 400, got %d", w.Code)
		}
	})

	t.Run("callback missing state", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/google/callback?code=abc", nil)
		r.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("expected: 400, got %d", w.Code)
		}
	})
}

func TestLogoutEndpoint(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RateLimiterRequestLimit = 1000
	r := testRouterFactory(t, testCfg, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest))
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("setup register failed, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("setup login failed, got %d", w.Code)
	}
	var login dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}

	// Create a second token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("setup second login failed, got %d", w.Code)
	}
	var login2 dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &login2); err != nil {
		t.Fatalf("failed to unmarshal second login response: %v", err)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/logout", nil)
	req.Header.Add("Authorization", "Bearer "+login.Token)
	r.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("expected: 204, got %d", w.Code)
	}

	// Token should be invalid after logout
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/validate", nil)
	req.Header.Add("Authorization", "Bearer "+login.Token)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected token to be invalid after logout, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/validate", nil)
	req.Header.Add("Authorization", "Bearer "+login2.Token)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected second token to be invalid after logout, got %d", w.Code)
	}
}

func TestDeleteUserEndpoint(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RedisURL = "redis"
	testCfg.IsRedisEnabled = true
	testCfg.RateLimiterRequestLimit = 1000
	r := testRouterFactory(t, testCfg, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest))
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("setup register failed, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("setup login failed, got %d", w.Code)
	}
	var login dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}

	// Create a second token
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("setup second login failed, got %d", w.Code)
	}
	var login2 dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &login2); err != nil {
		t.Fatalf("failed to unmarshal second login response: %v", err)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/me", nil)
	req.Header.Add("Authorization", "Bearer "+login.Token)
	r.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("expected: 204, got %d", w.Code)
	}

	// Token should be invalid after deletion
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/validate", nil)
	req.Header.Add("Authorization", "Bearer "+login.Token)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected token to be invalid after deletion, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/validate", nil)
	req.Header.Add("Authorization", "Bearer "+login2.Token)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected second token to be invalid after deletion, got %d", w.Code)
	}
}

func TestTwoFAEndpoints(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RedisURL = "redis"
	testCfg.IsRedisEnabled = true
	testCfg.RateLimiterRequestLimit = 1000
	r := testRouterFactory(t, testCfg, false)

	// setup user + login
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest))
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("setup register failed, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("setup login failed, got %d", w.Code)
	}
	var login dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}

	// 2FA setup
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/2fa/setup", nil)
	req.Header.Add("Authorization", "Bearer "+login.Token)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("2fa setup, expected: 200, got %d", w.Code)
	}
	var setup dto.TwoFASetupResponse
	if err := json.Unmarshal(w.Body.Bytes(), &setup); err != nil {
		t.Fatalf("failed to unmarshal 2fa setup response: %v", err)
	}
	twoFACode, err := totp.GenerateCode(setup.TwoFASecret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate 2fa code: %v", err)
	}

	// 2FA confirm invalid code
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/2fa/confirm", toJSON(t, map[string]string{
		"twoFaCode":  "000000",
		"setupToken": setup.SetupToken,
	}))
	req.Header.Add("Authorization", "Bearer "+login.Token)
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("2fa confirm invalid, expected: 400, got %d", w.Code)
	}

	// 2FA confirm happy
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/2fa/confirm", toJSON(t, map[string]string{
		"twoFaCode":  twoFACode,
		"setupToken": setup.SetupToken,
	}))
	req.Header.Add("Authorization", "Bearer "+login.Token)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("2fa confirm, expected: 200, got %d", w.Code)
	}

	// login now returns 428
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByUsernameRequest))
	r.ServeHTTP(w, req)
	if w.Code != 428 {
		t.Fatalf("login with 2fa enabled, expected: 428, got %d", w.Code)
	}
	var pending dto.TwoFAPendingUserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &pending); err != nil {
		t.Fatalf("failed to unmarshal 2fa pending response: %v", err)
	}

	// 2FA submit invalid code
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/2fa", toJSON(t, map[string]string{
		"twoFaCode":    "000000",
		"sessionToken": pending.SessionToken,
	}))
	r.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("2fa submit invalid, expected: 400, got %d", w.Code)
	}

	// 2FA submit happy
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/2fa", toJSON(t, map[string]string{
		"twoFaCode":    twoFACode,
		"sessionToken": pending.SessionToken,
	}))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("2fa submit, expected: 200, got %d", w.Code)
	}
	var afterChallenge dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &afterChallenge); err != nil {
		t.Fatalf("failed to unmarshal 2fa submit response: %v", err)
	}

	// Create a second token after 2FA enabled
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByUsernameRequest))
	r.ServeHTTP(w, req)
	if w.Code != 428 {
		t.Fatalf("login with 2fa enabled, expected: 428, got %d", w.Code)
	}
	var pending2 dto.TwoFAPendingUserResponse
	if err := json.Unmarshal(w.Body.Bytes(), &pending2); err != nil {
		t.Fatalf("failed to unmarshal 2fa pending response: %v", err)
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/2fa", toJSON(t, map[string]string{
		"twoFaCode":    twoFACode,
		"sessionToken": pending2.SessionToken,
	}))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("2fa submit second token, expected: 200, got %d", w.Code)
	}
	var afterChallenge2 dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &afterChallenge2); err != nil {
		t.Fatalf("failed to unmarshal 2fa submit second response: %v", err)
	}

	// 2FA disable wrong password
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/2fa/disable", toJSON(t, map[string]string{"password": "WrongPassword.123"}))
	req.Header.Add("Authorization", "Bearer "+afterChallenge.Token)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("2fa disable wrong password, expected: 401, got %d", w.Code)
	}

	// 2FA disable happy
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/2fa/disable", toJSON(t, map[string]string{"password": testPwd}))
	req.Header.Add("Authorization", "Bearer "+afterChallenge.Token)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("2fa disable, expected: 200, got %d", w.Code)
	}

	// Tokens should be invalid after 2FA disable
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/validate", nil)
	req.Header.Add("Authorization", "Bearer "+afterChallenge.Token)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected token to be invalid after 2fa disable, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/validate", nil)
	req.Header.Add("Authorization", "Bearer "+afterChallenge2.Token)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected second token to be invalid after 2fa disable, got %d", w.Code)
	}
}

func TestDBIsDown(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	r := testRouterFactory(t, testCfg, true)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest))
	r.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Fatalf("expected: 500, got: %d", w.Code)
	}
}

func TestAuthRequiredEndpoints(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RateLimiterRequestLimit = 1000
	r := testRouterFactory(t, testCfg, false)

	testCases := []struct {
		name   string
		method string
		path   string
	}{
		{name: "get me", method: http.MethodGet, path: "/me"},
		{name: "update password", method: http.MethodPut, path: "/password"},
		{name: "update profile", method: http.MethodPut, path: "/me"},
		{name: "logout", method: http.MethodDelete, path: "/logout"},
		{name: "delete me", method: http.MethodDelete, path: "/me"},
		{name: "2fa setup", method: http.MethodPost, path: "/2fa/setup"},
		{name: "2fa confirm", method: http.MethodPost, path: "/2fa/confirm"},
		{name: "2fa disable", method: http.MethodPut, path: "/2fa/disable"},
		{name: "get friends", method: http.MethodGet, path: "/friends"},
		{name: "add friend", method: http.MethodPost, path: "/friends"},
		{name: "validate user", method: http.MethodPost, path: "/validate"},
		{name: "list users", method: http.MethodGet, path: "/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"/no-token", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tc.method, tc.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != 401 {
				t.Fatalf("expected: 401, got %d", w.Code)
			}
		})

		t.Run(tc.name+"/invalid-token", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tc.method, tc.path, nil)
			req.Header.Add("Authorization", "Bearer aaa")
			r.ServeHTTP(w, req)

			if w.Code != 401 {
				t.Fatalf("expected: 401, got %d", w.Code)
			}
		})
	}
}

func TestValidationMissingBody(t *testing.T) {
	testCfg := testutil.NewTestConfig()
	testCfg.RateLimiterRequestLimit = 1000
	r := testRouterFactory(t, testCfg, false)

	// Create user and get a valid token for auth-required endpoints.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", toJSON(t, mockRegisterRequest))
	r.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Fatalf("registering user, expected: 201, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/loginByIdentifier", toJSON(t, mockLoginUserByEmailRequest))
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("login user, expected: 200, got %d", w.Code)
	}

	var login dto.UserWithTokenResponse
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}
	if login.Token == "" {
		t.Fatalf("login user, expected token to be set")
	}

	type validationErrorResp struct {
		Error []string `json:"error"`
	}

	assertValidationFields := func(t *testing.T, body []byte, expectedFields []string) {
		t.Helper()

		var resp validationErrorResp
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("failed to unmarshal validation error response: %v", err)
		}
		if len(resp.Error) == 0 {
			t.Fatalf("expected validation errors, got none")
		}

		for _, field := range expectedFields {
			found := false
			for _, msg := range resp.Error {
				if strings.Contains(msg, field) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected validation error to mention field %q, got: %v", field, resp.Error)
			}
		}
	}

	testCases := []struct {
		name           string
		method         string
		path           string
		needsAuth      bool
		expectedFields []string
	}{
		{name: "create user", method: http.MethodPost, path: "/", needsAuth: false, expectedFields: []string{"Username", "Email", "Password"}},
		{name: "login by identifier", method: http.MethodPost, path: "/loginByIdentifier", needsAuth: false, expectedFields: []string{"Identifier", "Password"}},
		{name: "2fa submit", method: http.MethodPost, path: "/2fa", needsAuth: false, expectedFields: []string{"TwoFACode", "SessionToken"}},
		{name: "update password", method: http.MethodPut, path: "/password", needsAuth: true, expectedFields: []string{"OldPassword", "NewPassword"}},
		{name: "update profile", method: http.MethodPut, path: "/me", needsAuth: true, expectedFields: []string{"Username", "Email"}},
		{name: "2fa confirm", method: http.MethodPost, path: "/2fa/confirm", needsAuth: true, expectedFields: []string{"TwoFACode", "SetupToken"}},
		{name: "2fa disable", method: http.MethodPut, path: "/2fa/disable", needsAuth: true, expectedFields: []string{"Password"}},
		{name: "add friend", method: http.MethodPost, path: "/friends", needsAuth: true, expectedFields: []string{"UserID"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			if tc.needsAuth {
				req.Header.Add("Authorization", "Bearer "+login.Token)
			}
			r.ServeHTTP(w, req)

			if w.Code != 400 {
				t.Fatalf("expected: 400, got %d", w.Code)
			}
			assertValidationFields(t, w.Body.Bytes(), tc.expectedFields)
		})
	}
}
