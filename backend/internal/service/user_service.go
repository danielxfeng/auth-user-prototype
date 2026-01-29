package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
	"github.com/redis/go-redis/v9"
)

const TwoFAPrePrefix = "pre-"
const BcryptSaltRounds = 10
const MaxAvatarSize = 1 * 1024 * 1024 // 1 MB
const BaseGoogleOAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"

type UserService struct {
	Dep  *dependency.Dependency
}

func (s *UserService) CreateUser(ctx context.Context, request *dto.CreateUserRequest) (*dto.UserWithoutTokenResponse, error) {

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

	err = gorm.G[model.User](s.Dep.DB).Create(ctx, &modelUser)
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

	modelUser, err := gorm.G[model.User](s.Dep.DB).Where(identifierField+" = ?", request.Identifier.Identifier).First(ctx)
	if err != nil || modelUser.PasswordHash == nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || modelUser.PasswordHash == nil {
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
		sessionToken, err := jwt.SignTwoFAToken(s.Dep, modelUser.ID)
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
	modelUser, err := gorm.G[model.User](s.Dep.DB).Where("id = ?", userID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(404, "user not found")
		}
		return nil, err
	}

	return userToUserWithoutTokenResponse(&modelUser), nil
}

func (s *UserService) UpdateUserPassword(ctx context.Context, userID uint, request *dto.UpdateUserPasswordRequest) (*dto.UserWithTokenResponse, error) {
	modelUser, err := gorm.G[model.User](s.Dep.DB).Where("id = ?", userID).First(ctx)
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

	_, err = gorm.G[model.User](s.Dep.DB).Where("id = ?", userID).Update(ctx, "password_hash", string(newPasswordBytes))
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
	modelUser, err := gorm.G[model.User](s.Dep.DB).Where("id = ?", userID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, middleware.NewAuthError(404, "user not found")
		}
		return nil, err
	}

	modelUser.Username = request.Username
	modelUser.Avatar = request.Avatar
	modelUser.Email = request.Email

	err = s.Dep.DB.WithContext(ctx).Save(&modelUser).Error

	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, middleware.NewAuthError(409, "username or email already in use")
		}
		return nil, err
	}

	return userToUserWithoutTokenResponse(&modelUser), nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID uint) error {
	if s.Dep.Cfg.IsRedisEnabled {
		err := logoutUserByRedis(ctx, s.Dep.Redis, userID)
		if err != nil {
			return err
		}
	}

	res := s.Dep.DB.WithContext(ctx).Unscoped().Delete(&model.User{}, userID)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

func logoutUserByDB(ctx context.Context, db *gorm.DB, userID uint) error {
	_, err := gorm.G[model.Token](db.Unscoped()).Where("user_id = ?", userID).Delete(ctx)
	if err != nil {
		return err
	}

	return nil
}

func logoutUserByRedis(ctx context.Context, redis *redis.Client, userID uint) error {

	iter := redis.Scan(ctx, 0, buildTokenKey(userID, "*"), 100).Iterator()
	for iter.Next(ctx) {
		err := redis.Del(ctx, iter.Val()).Err()
		if err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}

	return nil
}

func (s *UserService) LogoutUser(ctx context.Context, userID uint) error {
	if s.Dep.Cfg.IsRedisEnabled {
		return logoutUserByRedis(ctx, s.Dep.Redis, userID)
	} else {
		return logoutUserByDB(ctx, s.Dep.DB, userID)
	}
}

func (s *UserService) validateUserTokenDB(ctx context.Context, token string, userId uint) error {
	modelToken, err := gorm.G[model.Token](s.Dep.DB).Where("token = ?", token).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return middleware.NewAuthError(401, "invalid token")
		}
		return err
	}

	if modelToken.UserID != userId {
		return middleware.NewAuthError(401, "token does not match user")
	}

	s.updateHeartBeat(userId)
	return nil
}

func (s *UserService) validateUserTokenRedis(ctx context.Context, token string, userId uint) error {
	_, err := s.Dep.Redis.Get(ctx, buildTokenKey(userId, token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return middleware.NewAuthError(401, "invalid token")
		}
		return err
	}

	// A rough way to implement sliding expiration
	s.Dep.Redis.Expire(ctx, buildTokenKey(userId, token), time.Duration(s.Dep.Cfg.UserTokenExpiry)*time.Second)

	s.updateHeartBeat(userId)
	return nil
}

func (s *UserService) ValidateUserToken(ctx context.Context, token string, userId uint) error {
	if s.Dep.Cfg.IsRedisEnabled {
		return s.validateUserTokenRedis(ctx, token, userId)
	} else {
		return s.validateUserTokenDB(ctx, token, userId)
	}
}
