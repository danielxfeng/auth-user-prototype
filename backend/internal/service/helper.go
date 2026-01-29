package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const HeartBeatPrefix = "heartbeat:"

func NewUserService(dep *dependency.Dependency) *UserService {

	if dep.DB == nil {
		panic("UserService: db is nil")
	}

	if dep.Cfg.IsRedisEnabled && dep.Redis == nil {
		panic("UserService: redis is enabled but redis client is nil")
	}

	return &UserService{
		Dep:  dep,
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

func (s *UserService) updateHeartBeatByDB(userID uint) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := s.Dep.DB.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			UpdateAll: true,
		}).Create(&model.HeartBeat{
			UserID:     userID,
			LastSeenAt: time.Now(),
		}).Error

		if err != nil {
			s.Dep.Logger.Warn("failed to update heartbeat for user", fmt.Sprint(userID), err.Error())
		}
	}()
}

func (s *UserService) updateHeartBeatByRedis(userID uint) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := s.Dep.Redis.ZAdd(ctx, HeartBeatPrefix, redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: userID,
		}).Err()

		if err != nil {
			s.Dep.Logger.Warn("failed to update heartbeat for user", fmt.Sprint(userID), err.Error())
		}
	}()
}

func (s *UserService) updateHeartBeat(userID uint) {
	if s.Dep.Cfg.IsRedisEnabled {
		s.updateHeartBeatByRedis(userID)
	} else {
		s.updateHeartBeatByDB(userID)
	}
}

func (s *UserService) getOnlineStatusByDB(ctx context.Context) ([]model.HeartBeat, error) {
	onlineStatus, err := gorm.G[model.HeartBeat](s.Dep.DB).Where("last_seen_at > ?", time.Now().Add(-2*time.Minute)).Find(ctx)
	if err != nil {
		return nil, err
	}

	return onlineStatus, nil
}

func (s *UserService) clearExpiredHeartBeatsByRedis() {

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := s.Dep.Redis.ZRemRangeByScore(ctx, HeartBeatPrefix, "-inf", strconv.FormatInt(time.Now().Add(-2*time.Minute).Unix(), 10)).Err()
		if err != nil {
			s.Dep.Logger.Warn("failed to clear expired heartbeats from redis", "err", err)
		}
	}()
}

func (s *UserService) getOnlineStatusByRedis(ctx context.Context) ([]model.HeartBeat, error) {
	heartBeats := make([]model.HeartBeat, 0)

	zs, err := s.Dep.Redis.ZRangeByScoreWithScores(ctx, HeartBeatPrefix, &redis.ZRangeBy{
		Min: strconv.FormatInt(time.Now().Add(-2*time.Minute).Unix(), 10),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, err
	}

	for _, z := range zs {
		userID, err := strconv.ParseUint(fmt.Sprint(z.Member), 10, 64)
		if err != nil {
			return nil, err
		}

		heartBeats = append(heartBeats, model.HeartBeat{
			UserID:     uint(userID),
			LastSeenAt: time.Unix(int64(z.Score), 0),
		})
	}

	// Async clear expired heartbeats
	s.clearExpiredHeartBeatsByRedis()

	return heartBeats, nil
}

func (s *UserService) getOnlineStatus(ctx context.Context) ([]model.HeartBeat, error) {
	if s.Dep.Cfg.IsRedisEnabled {
		return s.getOnlineStatusByRedis(ctx)
	} else {
		return s.getOnlineStatusByDB(ctx)
	}
}

func buildTokenKey(userID uint, token string) string {
	return fmt.Sprintf("user_token:%d:%s", userID, token)
}

func (s *UserService) issueNewTokenForUserByDB(ctx context.Context, userID uint, revokeAllTokens bool) (string, error) {

	if revokeAllTokens {
		res := s.Dep.DB.WithContext(ctx).Exec("DELETE FROM tokens WHERE user_id = ?", userID)
		if res.Error != nil {
			return "", res.Error
		}
	}

	token, err := jwt.SignUserToken(s.Dep, userID)
	if err != nil {
		return "", err
	}

	err = gorm.G[model.Token](s.Dep.DB).Create(ctx, &model.Token{
		UserID: userID,
		Token:  token,
	})
	if err != nil {
		return "", err
	}

	s.updateHeartBeat(userID)

	return token, nil
}

func (s *UserService) issueNewTokenForUserByRedis(ctx context.Context, userID uint, revokeAllTokens bool) (string, error) {

	if revokeAllTokens {
		// A rough way to delete all tokens for the user
		iter := s.Dep.Redis.Scan(ctx, 0, buildTokenKey(userID, "*"), 100).Iterator()
		for iter.Next(ctx) {
			err := s.Dep.Redis.Del(ctx, iter.Val()).Err()
			if err != nil {
				return "", err
			}
		}
		if err := iter.Err(); err != nil {
			return "", err
		}
	}

	token, err := jwt.SignUserToken(s.Dep, userID)
	if err != nil {
		return "", err
	}

	err = s.Dep.Redis.Set(ctx, buildTokenKey(userID, token), "", time.Duration(s.Dep.Cfg.UserTokenExpiry)*time.Second).Err()
	if err != nil {
		return "", err
	}

	s.updateHeartBeat(userID)

	return token, nil
}

func (s *UserService) issueNewTokenForUser(ctx context.Context, userID uint, revokeAllTokens bool) (string, error) {
	if s.Dep.Cfg.IsRedisEnabled {
		return s.issueNewTokenForUserByRedis(ctx, userID, revokeAllTokens)
	} else {
		return s.issueNewTokenForUserByDB(ctx, userID, revokeAllTokens)
	}
}
