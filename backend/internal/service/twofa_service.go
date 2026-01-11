package service

import (
	"context"
	"errors"
	"strings"

	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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
		TwoFaUri:    secret.URL(),
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
	modelUser.TwoFAToken = &twoFaSecret

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
	modelUser.TwoFAToken = nil

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
