package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
	"gorm.io/gorm"
)

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

func (s *UserService) updateHeartBeat(userID uint) {
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

	s.updateHeartBeat(userID)

	return token, nil
}