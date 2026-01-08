package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
	"google.golang.org/api/idtoken"
)

const TwoFAPrePrefix = "pre-"
const BcryptSaltRounds = 10
const MaxAvatarSize = 1 * 1024 * 1024 // 1 MB
const BaseGoogleOAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"

type UserService struct {
	DB *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		DB: db,
	}
}

func isTwoFAEnabled(twoFAToken *string) bool {
	return twoFAToken != nil && *twoFAToken != "" && !strings.HasPrefix(*twoFAToken, TwoFAPrePrefix)
}

func userToUserWithoutTokenResponse(user *model.User) *dto.UserWithoutTokenResponse {
	isTwoFAEnabled := isTwoFAEnabled(user.TwoFAToken)

	return &dto.UserWithoutTokenResponse{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		Avatar:        user.Avatar,
		TwoFA:         isTwoFAEnabled,
		GoogleOauthId: user.GoogleOauthID,
		CreatedAt:     user.CreatedAt.Unix(),
	}
}

func userToUserWithTokenResponse(user *model.User, token string) *dto.UserWithTokenResponse {
	isTwoFAEnabled := isTwoFAEnabled(user.TwoFAToken)

	return &dto.UserWithTokenResponse{
		ID:            user.ID,
		Username:      user.Username,
		Email:         user.Email,
		Avatar:        user.Avatar,
		TwoFA:         isTwoFAEnabled,
		GoogleOauthId: user.GoogleOauthID,
		CreatedAt:     user.CreatedAt.Unix(),
		Token:         token,
	}
}

func userToSimpleUser(user *model.User) *dto.SimpleUser {
	return &dto.SimpleUser{
		ID:       user.ID,
		Username: user.Username,
		Avatar:   user.Avatar,
	}
}

func (s *UserService) UpdateHeartBeat(userID uint) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := gorm.G[model.HeartBeat](s.DB).Create(ctx, &model.HeartBeat{
			UserID: userID,
		})

		if err != nil {
			util.Logger.Warn("failed to update heartbeat for user", fmt.Sprint(userID), err.Error())
		}
	}()
}

func (s *UserService) issueNewTokenForUser(ctx context.Context, userID uint, revokeAllTokens bool) (string, error) {

	if revokeAllTokens {
		_, err := gorm.G[model.Token](s.DB).Where("user_id = ?", userID).Delete(ctx)
		if err != nil {
			return "", err
		}
	}

	token, err := jwt.SignUserToken(userID)
	if err != nil {
		return "", err
	}

	err = gorm.G[model.Token](s.DB).Create(ctx, &model.Token{
		UserID: userID,
		Token:  token,
	})
	if err != nil {
		return "", err
	}

	s.UpdateHeartBeat(userID)

	return token, nil
}

func validateAvatarURL(avatar *string, maxSize int) bool {
	if avatar == nil {
		return true
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequest(http.MethodHead, *avatar, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") || ct == "image/svg+xml" {
		return false
	}

	cl := resp.Header.Get("Content-Length")
	if cl == "" {
		return false
	}

	size, err := strconv.Atoi(cl)
	if err != nil || size <= 0 || size > maxSize {
		return false
	}

	return true
}

func (s *UserService) CreateUser(ctx context.Context, request *dto.CreateUserRequest) (*dto.UserWithoutTokenResponse, error) {

	if !validateAvatarURL(request.Avatar, MaxAvatarSize) {
		return nil, middleware.NewAuthError(400, "invalid avatar URL or avatar size exceeds limit")
	}

	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(request.Password.Password), BcryptSaltRounds)
	if err != nil {
		return nil, err
	}

	passwordHash := string(passwordBytes)

	modelUser := model.User{
		Username:      request.Username,
		Email:         request.Email,
		PasswordHash:  &passwordHash,
		Avatar:        request.Avatar,
		GoogleOauthID: nil,
		TwoFAToken:    nil,
	}

	err = gorm.G[model.User](s.DB).Create(ctx, &modelUser)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, middleware.NewAuthError(409, "username or email already in use")
		}
		return nil, err
	}

	return userToUserWithoutTokenResponse(&modelUser), nil
}

type LoginResult struct {
	User         *dto.UserWithTokenResponse
	TwoFAPending *dto.TwoFAPendingUserResponse
}

func (s *UserService) LoginUser(ctx context.Context, request *dto.LoginUserRequest) (*LoginResult, error) {

	var identifierField string
	if strings.Contains(request.Identifier.Identifier, "@") {
		identifierField = "email"
	} else {
		identifierField = "username"
	}

	modelUser, err := gorm.G[model.User](s.DB).Where(identifierField+" = ?", request.Identifier.Identifier).First(ctx)
	if err != nil || modelUser.PasswordHash == nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(401, "invalid credentials")
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(*modelUser.PasswordHash), []byte(request.Password.Password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, middleware.NewAuthError(401, "invalid credentials")
		}
		return nil, err
	}

	isTwoFAEnabled := isTwoFAEnabled(modelUser.TwoFAToken)
	if isTwoFAEnabled {
		sessionToken, err := jwt.SignTwoFAToken(modelUser.ID)
		if err != nil {
			return nil, err
		}

		return &LoginResult{
			TwoFAPending: &dto.TwoFAPendingUserResponse{
				Message:      "2FA_REQUIRED",
				SessionToken: sessionToken,
			},
		}, nil
	}

	userToken, err := s.issueNewTokenForUser(ctx, modelUser.ID, false)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User: userToUserWithTokenResponse(&modelUser, userToken),
	}, nil
}

func (s *UserService) GetUserByID(ctx context.Context, userID uint) (*dto.UserWithoutTokenResponse, error) {
	modelUser, err := gorm.G[model.User](s.DB).Where("id = ?", userID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(404, "user not found")
		}
		return nil, err
	}

	return userToUserWithoutTokenResponse(&modelUser), nil
}

func (s *UserService) UpdateUserPassword(ctx context.Context, userID uint, request *dto.UpdateUserPasswordRequest) (*dto.UserWithTokenResponse, error) {
	modelUser, err := gorm.G[model.User](s.DB).Where("id = ?", userID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(404, "user not found")
		}
		return nil, err
	}

	if modelUser.PasswordHash == nil {
		return nil, middleware.NewAuthError(400, "password cannot be changed for OAuth users")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*modelUser.PasswordHash), []byte(request.OldPassword.OldPassword))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, middleware.NewAuthError(401, "invalid credentials")
		}
		return nil, err
	}

	newPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword.NewPassword), BcryptSaltRounds)
	if err != nil {
		return nil, err
	}

	_, err = gorm.G[model.User](s.DB).Where("id = ?", userID).Update(ctx, "password_hash", string(newPasswordBytes))
	if err != nil {
		return nil, err
	}

	userToken, err := s.issueNewTokenForUser(ctx, userID, true)
	if err != nil {
		return nil, err
	}

	return userToUserWithTokenResponse(&modelUser, userToken), nil
}

func (s *UserService) UpdateUserProfile(ctx context.Context, userID uint, request *dto.UpdateUserRequest) (*dto.UserWithoutTokenResponse, error) {
	modelUser, err := gorm.G[model.User](s.DB).Where("id = ?", userID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(404, "user not found")
		}
		return nil, err
	}

	if !validateAvatarURL(request.Avatar, MaxAvatarSize) {
		return nil, middleware.NewAuthError(400, "invalid avatar URL or avatar size exceeds limit")
	}

	modelUser.Username = request.Username
	modelUser.Avatar = request.Avatar
	modelUser.Email = request.Email

	err = s.DB.WithContext(ctx).Save(&modelUser).Error

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, middleware.NewAuthError(409, "username or email already in use")
		}
		return nil, err
	}

	return userToUserWithoutTokenResponse(&modelUser), nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID uint) error {
	_, err := gorm.G[model.Token](s.DB).Where("user_id = ?", userID).Delete(ctx)
	if err != nil {
		return err
	}

	_, err = gorm.G[model.Friend](s.DB).Where("user_id = ? OR friend_id = ?", userID, userID).Delete(ctx)
	if err != nil {
		return err
	}

	_, err = gorm.G[model.HeartBeat](s.DB).Where("user_id = ?", userID).Delete(ctx)
	if err != nil {
		return err
	}

	_, err = gorm.G[model.User](s.DB).Where("id = ?", userID).Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) StartTwoFaSetup(ctx context.Context, userID uint) (*dto.TwoFASetupResponse, error) {
	modelUser, err := gorm.G[model.User](s.DB).Where("id = ?", userID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(404, "user not found")
		}
		return nil, err
	}

	if isTwoFAEnabled(modelUser.TwoFAToken) {
		return nil, middleware.NewAuthError(400, "2FA is already enabled")
	}

	if modelUser.GoogleOauthID != nil {
		return nil, middleware.NewAuthError(400, "2FA cannot be enabled for Google OAuth users")
	}

	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Transcendence",
		AccountName: modelUser.Email,
	})
	if err != nil {
		return nil, err
	}

	twoFAToken := TwoFAPrePrefix + secret.Secret()

	_, err = gorm.G[model.User](s.DB).Where("id = ?", userID).Update(ctx, "two_fa_token", twoFAToken)
	if err != nil {
		return nil, err
	}

	setupToken, err := jwt.SignTwoFASetupToken(userID, secret.Secret())
	if err != nil {
		return nil, err
	}

	return &dto.TwoFASetupResponse{
		TwoFASecret: secret.Secret(),
		SetupToken:  setupToken,
	}, nil
}

func (s *UserService) ConfirmTwoFaSetup(ctx context.Context, userID uint, request *dto.TwoFAConfirmRequest) (*dto.UserWithTokenResponse, error) {
	claims, err := jwt.ValidateTwoFASetupToken(request.SetupToken)
	if err != nil || claims.Type != jwt.TwoFASetupType {
		return nil, middleware.NewAuthError(400, "invalid setup token")
	}

	if claims.UserID != userID {
		return nil, middleware.NewAuthError(400, "setup token does not match user")
	}

	modelUser, err := gorm.G[model.User](s.DB).Where("id = ?", userID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(404, "user not found")
		}
		return nil, err
	}

	if modelUser.TwoFAToken == nil {
		return nil, middleware.NewAuthError(400, "2FA setup was not initiated")
	}

	if isTwoFAEnabled(modelUser.TwoFAToken) {
		return nil, middleware.NewAuthError(400, "2FA is already enabled")
	}

	if modelUser.GoogleOauthID != nil {
		return nil, middleware.NewAuthError(400, "2FA cannot be enabled for Google OAuth users")
	}

	twoFaSecret := strings.TrimPrefix(*modelUser.TwoFAToken, TwoFAPrePrefix)
	valid := totp.Validate(request.TwoFACode, twoFaSecret)
	if !valid {
		return nil, middleware.NewAuthError(400, "invalid 2FA code")
	}

	_, err = gorm.G[model.User](s.DB).Where("id = ?", userID).Update(ctx, "two_fa_token", twoFaSecret)
	if err != nil {
		return nil, err
	}

	userToken, err := s.issueNewTokenForUser(ctx, userID, true)
	if err != nil {
		return nil, err
	}

	return userToUserWithTokenResponse(&modelUser, userToken), nil
}

func (s *UserService) DisableTwoFA(ctx context.Context, userID uint, request *dto.DisableTwoFARequest) (*dto.UserWithTokenResponse, error) {
	modelUser, err := gorm.G[model.User](s.DB).Where("id = ?", userID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(404, "user not found")
		}
		return nil, err
	}

	if modelUser.PasswordHash == nil {
		return nil, middleware.NewAuthError(400, "2FA cannot be disabled for OAuth users")
	}

	if !isTwoFAEnabled(modelUser.TwoFAToken) {
		return nil, middleware.NewAuthError(400, "2FA is not enabled")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*modelUser.PasswordHash), []byte(request.Password.Password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, middleware.NewAuthError(401, "invalid credentials")
		}
		return nil, err
	}

	_, err = gorm.G[model.User](s.DB).Where("id = ?", userID).Update(ctx, "two_fa_token", nil)
	if err != nil {
		return nil, err
	}

	userToken, err := s.issueNewTokenForUser(ctx, userID, true)
	if err != nil {
		return nil, err
	}

	return userToUserWithTokenResponse(&modelUser, userToken), nil
}

func (s *UserService) SubmitTwoFAChallenge(ctx context.Context, request *dto.TwoFAChallengeRequest) (*dto.UserWithTokenResponse, error) {
	claims, err := jwt.ValidateTwoFAToken(request.SessionToken)
	if err != nil || claims.Type != jwt.TwoFATokenType {
		return nil, middleware.NewAuthError(400, "invalid session token")
	}

	modelUser, err := gorm.G[model.User](s.DB).Where("id = ?", claims.UserID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(404, "user not found")
		}
		return nil, err
	}

	if !isTwoFAEnabled(modelUser.TwoFAToken) || modelUser.TwoFAToken == nil {
		return nil, middleware.NewAuthError(400, "2FA is not enabled for this user")
	}

	valid := totp.Validate(request.TwoFACode, *modelUser.TwoFAToken)
	if !valid {
		return nil, middleware.NewAuthError(400, "invalid 2FA code")
	}

	userToken, err := s.issueNewTokenForUser(ctx, modelUser.ID, false)
	if err != nil {
		return nil, err
	}

	return userToUserWithTokenResponse(&modelUser, userToken), nil
}

func (s *UserService) GetAllUsersLimitedInfo(ctx context.Context) (*dto.UsersResponse, error) {
	modelUsers, err := gorm.G[model.User](s.DB).Find(ctx)
	if err != nil {
		return nil, err
	}

	simpleUsers := make([]dto.SimpleUser, 0, len(modelUsers))
	for _, mu := range modelUsers {
		simpleUsers = append(simpleUsers, *userToSimpleUser(&mu))
	}

	return &dto.UsersResponse{
		Users: simpleUsers,
	}, nil
}

type onlineStatusChecker struct {
	heartBeatSet map[uint]struct{}
}

func newOnlineStatusChecker(heartBeats []model.HeartBeat) *onlineStatusChecker {
	hs := &onlineStatusChecker{
		heartBeatSet: make(map[uint]struct{}, len(heartBeats)),
	}

	for _, hb := range heartBeats {
		hs.heartBeatSet[hb.UserID] = struct{}{}
	}

	return hs
}

func (os *onlineStatusChecker) isOnline(userID uint) bool {
	_, exists := os.heartBeatSet[userID]
	return exists
}

func (s *UserService) GetUserFriends(ctx context.Context, userID uint) (*dto.FriendsResponse, error) {
	friends, err := gorm.G[model.Friend](s.DB).Preload("Friend", nil).Where("user_id = ?", userID).Find(ctx)
	if err != nil {
		return nil, err
	}

	onlineStatus, err := gorm.G[model.HeartBeat](s.DB).Where("created_at > ?", time.Now().Add(-5*time.Minute)).Find(ctx)
	if err != nil {
		return nil, err
	}

	checker := newOnlineStatusChecker(onlineStatus)

	friendResponses := make([]dto.FriendResponse, 0, len(friends))
	for _, f := range friends {
		friendResponses = append(friendResponses, dto.FriendResponse{
			SimpleUser: *userToSimpleUser(&f.Friend),
			Online:     checker.isOnline(f.FriendID),
		})
	}

	return &dto.FriendsResponse{
		Friends: friendResponses,
	}, nil
}

func (s *UserService) AddNewFriend(ctx context.Context, userID uint, request *dto.AddNewFriendRequest) error {

	if userID == request.UserID {
		return middleware.NewAuthError(400, "cannot add yourself as a friend")
	}

	newFriend := model.Friend{
		UserID:   userID,
		FriendID: request.UserID,
	}

	err := gorm.G[model.Friend](s.DB).Create(ctx, &newFriend)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return middleware.NewAuthError(409, "friend already added")
		}
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			return middleware.NewAuthError(404, "user not found")
		}
		return err
	}

	return nil
}

func (s *UserService) GetGoogleOAuthURL(ctx context.Context) (string, error) {
	state, err := jwt.SignOauthStateToken()
	if err != nil {
		util.Logger.Error("failed to sign oauth state token:", "err", err)
		return "", err
	}

	u, err := url.Parse(BaseGoogleOAuthURL)
	if err != nil {
		util.Logger.Error("failed to parse google oauth base url:", "err", err)
		return "", err
	}

	q := u.Query()
	q.Set("client_id", config.Cfg.GoogleClientId)
	q.Set("redirect_uri", config.Cfg.GoogleRedirectUri)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)

	u.RawQuery = q.Encode()

	return u.String(), nil
}

func assembleFrontendRedirectURL(token *string, errMsg *string) string {
	u, err := url.Parse(config.Cfg.FrontendUrl + "/oauth/callback")
	if err != nil {
		util.Logger.Error("failed to parse frontend redirect url:", "err", err)
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

func (s *UserService) exchangeCodeForTokens(ctx context.Context, code string) (*idtoken.Payload, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", config.Cfg.GoogleClientId)
	data.Set("client_secret", config.Cfg.GoogleClientSecret)
	data.Set("redirect_uri", config.Cfg.GoogleRedirectUri)
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

	payload, err := idtoken.Validate(ctx, tokenResp.IdToken, config.Cfg.GoogleClientId)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func fetchGoogleUserInfo(payload *idtoken.Payload) (*dto.GoogleUserData, error) {
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

func (s *UserService) linkGoogleAccountToExistingUser(ctx context.Context, modelUser *model.User, googleUserInfo *dto.GoogleUserData) error {

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
}

func (s *UserService) createNewUserFromGoogleInfo(ctx context.Context, googleUserInfo *dto.GoogleUserData, isRetry bool) (*model.User, error) {

	username := ""

	if isRetry {
		uuidUsername, err := uuid.NewRandom()
		if err != nil {
			return nil, middleware.NewAuthError(500, "failed to generate UUID for Google user")
		}
		username = "google_" + uuidUsername.String()
	} else {
		username = "google_" + googleUserInfo.ID
	}

	modelUser := model.User{
		Username:      username,
		Email:         googleUserInfo.Email,
		PasswordHash:  nil,
		Avatar:        googleUserInfo.Picture,
		GoogleOauthID: &googleUserInfo.ID,
		TwoFAToken:    nil,
	}

	err := gorm.G[model.User](s.DB).Create(ctx, &modelUser)
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

func HandleGoogleOAuthCallbackError(err error, errMsg string) string {
	publicMsg := "Failed to handle Google OAuth callback."
	util.Logger.Error(errMsg, "error", err)
	return assembleFrontendRedirectURL(nil, &publicMsg)
}

func (s *UserService) HandleGoogleOAuthCallback(ctx context.Context, code string, state string) string {
	var finalUserID uint

	claims, err := jwt.ValidateOauthStateToken(state)
	if err != nil || claims.Type != jwt.GoogleOAuthStateType {
		return HandleGoogleOAuthCallbackError(err, "invalid oauth state token")
	}

	googlePayload, err := s.exchangeCodeForTokens(ctx, code)
	if err != nil {
		return HandleGoogleOAuthCallbackError(err, "failed to exchange code for tokens")
	}

	googleUserInfo, err := fetchGoogleUserInfo(googlePayload)
	if err != nil {
		return HandleGoogleOAuthCallbackError(err, "failed to fetch google user info from id token")
	}

	modelUser, err := gorm.G[model.User](s.DB).Where("google_oauth_id = ?", googleUserInfo.ID).First(ctx)
	if err == nil { // User with this Google OAuth ID exists, log them in
		finalUserID = modelUser.ID
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return HandleGoogleOAuthCallbackError(err, "failed to query user by google oauth id")
	} else {
		// No user with this Google OAuth ID, check if a user with this email exists
		modelUser, err = gorm.G[model.User](s.DB).Where("email = ?", googleUserInfo.Email).First(ctx)
		if err == nil { // User with this email exists, link Google account

			err = s.linkGoogleAccountToExistingUser(ctx, &modelUser, googleUserInfo)
			if err != nil { // Failed to link Google account
				return HandleGoogleOAuthCallbackError(err, "failed to link google account to existing user")
			}
			// Successfully linked Google account
			finalUserID = modelUser.ID
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return HandleGoogleOAuthCallbackError(err, "failed to query user by email")
		} else {
			// No user with this email exists, create a new user
			newUser, err := s.createNewUserFromGoogleInfo(ctx, googleUserInfo, false)
			if err != nil {
				return HandleGoogleOAuthCallbackError(err, "failed to create new user from google info")
			}

			finalUserID = newUser.ID
		}
	}

	if finalUserID == 0 {
		return HandleGoogleOAuthCallbackError(errors.New("finalUserID is zero"), "internal error determining final user ID")
	}

	userToken, err := s.issueNewTokenForUser(ctx, finalUserID, false)
	if err != nil {
		return HandleGoogleOAuthCallbackError(err, "failed to issue new token for user")
	}

	return assembleFrontendRedirectURL(&userToken, nil)
}
