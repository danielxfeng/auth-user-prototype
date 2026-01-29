package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/auth/credentials/idtoken"
	"github.com/google/uuid"
	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
	"gorm.io/gorm"
)

func (s *UserService) GetGoogleOAuthURL(ctx context.Context) (string, error) {
	state, err := jwt.SignOauthStateToken(s.Dep)
	if err != nil {
		s.Dep.Logger.Error("failed to sign oauth state token:", "err", err)
		return "", err
	}

	u, err := url.Parse(BaseGoogleOAuthURL)
	if err != nil {
		s.Dep.Logger.Error("failed to parse google oauth base url:", "err", err)
		return "", err
	}

	q := u.Query()
	q.Set("client_id", s.Dep.Cfg.GoogleClientId)
	q.Set("redirect_uri", s.Dep.Cfg.GoogleRedirectUri)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)

	u.RawQuery = q.Encode()

	return u.String(), nil
}

func assembleFrontendRedirectURL(dep *dependency.Dependency, token *string, errMsg *string) string {
	u, err := url.Parse(dep.Cfg.FrontendUrl + "/user/oauth-callback-google")
	if err != nil {
		dep.Logger.Error("failed to parse frontend redirect url:", "err", err)
		return "/unrecovered-error"
	}

	q := u.Query()
	if token != nil {
		q.Set("token", *token)
	}
	if errMsg != nil {
		q.Set("error", *errMsg)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

var ExchangeCodeForTokens = func(dep *dependency.Dependency, ctx context.Context, code string) (*idtoken.Payload, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", dep.Cfg.GoogleClientId)
	data.Set("client_secret", dep.Cfg.GoogleClientSecret)
	data.Set("redirect_uri", dep.Cfg.GoogleRedirectUri)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to exchange code for tokens")
	}

	tokenResp := dto.GoogleJwtPayload{}

	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return nil, err
	}

	payload, err := idtoken.Validate(ctx, tokenResp.IdToken, dep.Cfg.GoogleClientId)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

var FetchGoogleUserInfo = func(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
	sub := payload.Subject
	if sub == "" {
		return nil, middleware.NewAuthError(400, "google id token missing subject")
	}

	jsonClaims, err := json.Marshal(payload.Claims)
	if err != nil {
		return nil, middleware.NewAuthError(500, "failed to Marshal google jwt token")
	}

	var claims dto.GoogleClaims
	err = json.Unmarshal(jsonClaims, &claims)
	if err != nil {
		return nil, middleware.NewAuthError(500, "failed to Unmarshal google jwt token")
	}

	googleUserInfo := &dto.GoogleUserData{
		ID:    sub,
		Email: claims.Email,
		Name:  claims.Name,
	}

	if claims.Picture != "" {
		googleUserInfo.Picture = &claims.Picture
	}

	return googleUserInfo, nil
}

// This feature does not work unless we can verify the user's password/email ownership.
func (s *UserService) linkGoogleAccountToExistingUser(ctx context.Context, modelUser *model.User, googleUserInfo *dto.GoogleUserData) error {
	return middleware.NewAuthError(409, "same email exists")

	/**

	// Should not be here.
	if modelUser.Email != googleUserInfo.Email {
		return middleware.NewAuthError(500, "email mismatch between existing account and Google account")
	}

	if modelUser.GoogleOauthID != nil {
		return middleware.NewAuthError(400, "user already has a linked Google account")
	}

	if isTwoFAEnabled(modelUser.TwoFAToken) {
		return middleware.NewAuthError(400, "cannot link Google account when 2FA is enabled")
	}

	modelUser.GoogleOauthID = &googleUserInfo.ID
	if googleUserInfo.Picture != nil {
		modelUser.Avatar = googleUserInfo.Picture
	}

	err := s.DB.WithContext(ctx).Save(modelUser).Error
	if err != nil {
		return err
	}

	return nil
	*/
}

func (s *UserService) createNewUserFromGoogleInfo(ctx context.Context, googleUserInfo *dto.GoogleUserData, isRetry bool) (*model.User, error) {

	username := ""

	if isRetry {
		uuidUsername, err := uuid.NewRandom()
		if err != nil {
			return nil, middleware.NewAuthError(500, "failed to generate UUID for Google user")
		}
		username = "G_" + uuidUsername.String()
	} else {
		idPrefix := googleUserInfo.ID
		if len(idPrefix) > 8 {
			idPrefix = idPrefix[:8]
		}
		username = "G_" + idPrefix
	}

	modelUser := model.User{
		Username:      username,
		Email:         googleUserInfo.Email,
		PasswordHash:  nil,
		Avatar:        googleUserInfo.Picture,
		GoogleOauthID: &googleUserInfo.ID,
		TwoFAToken:    nil,
	}

	err := gorm.G[model.User](s.Dep.DB).Create(ctx, &modelUser)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			if !isRetry {
				return s.createNewUserFromGoogleInfo(ctx, googleUserInfo, true)
			}
			return nil, middleware.NewAuthError(409, "username or email already in use")
		}
		return nil, err
	}

	return &modelUser, nil
}

func HandleGoogleOAuthCallbackError(dep *dependency.Dependency, err error, errMsg string) string {
	publicMsg := "Failed to handle Google OAuth callback."
	dep.Logger.Error(errMsg, "error", err)
	return assembleFrontendRedirectURL(dep, nil, &publicMsg)
}

func (s *UserService) HandleGoogleOAuthCallback(ctx context.Context, code string, state string) string {
	var finalUserID uint

	claims, err := jwt.ValidateOauthStateToken(s.Dep, state)
	if err != nil || claims.Type != jwt.GoogleOAuthStateType {
		return HandleGoogleOAuthCallbackError(s.Dep, err, "invalid oauth state token")
	}

	googlePayload, err := ExchangeCodeForTokens(s.Dep, ctx, code)
	if err != nil {
		return HandleGoogleOAuthCallbackError(s.Dep, err, "failed to exchange code for tokens")
	}

	googleUserInfo, err := FetchGoogleUserInfo(googlePayload)
	if err != nil {
		return HandleGoogleOAuthCallbackError(s.Dep, err, "failed to fetch google user info from id token")
	}

	modelUser, err := gorm.G[model.User](s.Dep.DB).Where("google_oauth_id = ?", googleUserInfo.ID).First(ctx)
	if err == nil { // User with this Google OAuth ID exists, log them in
		finalUserID = modelUser.ID
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return HandleGoogleOAuthCallbackError(s.Dep, err, "failed to query user by google oauth id")
	} else {
		// No user with this Google OAuth ID, check if a user with this email exists
		modelUser, err = gorm.G[model.User](s.Dep.DB).Where("email = ?", googleUserInfo.Email).First(ctx)
		if err == nil { // User with this email exists, link Google account

			err = s.linkGoogleAccountToExistingUser(ctx, &modelUser, googleUserInfo)
			if err != nil { // Failed to link Google account
				return HandleGoogleOAuthCallbackError(s.Dep, err, "failed to link google account to existing user")
			}
			// Successfully linked Google account
			finalUserID = modelUser.ID
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return HandleGoogleOAuthCallbackError(s.Dep, err, "failed to query user by email")
		} else {
			// No user with this email exists, create a new user
			newUser, err := s.createNewUserFromGoogleInfo(ctx, googleUserInfo, false)
			if err != nil {
				return HandleGoogleOAuthCallbackError(s.Dep, err, "failed to create new user from google info")
			}

			finalUserID = newUser.ID
		}
	}

	if finalUserID == 0 {
		return HandleGoogleOAuthCallbackError(s.Dep, errors.New("finalUserID is zero"), "internal error determining final user ID")
	}

	userToken, err := s.issueNewTokenForUser(ctx, finalUserID, false)
	if err != nil {
		return HandleGoogleOAuthCallbackError(s.Dep, err, "failed to issue new token for user")
	}

	return assembleFrontendRedirectURL(s.Dep, &userToken, nil)
}
