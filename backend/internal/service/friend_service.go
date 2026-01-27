package service

import (
	"context"
	"errors"

	model "github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"gorm.io/gorm"
)

func (s *UserService) GetAllUsersLimitedInfo(ctx context.Context) ([]dto.SimpleUser, error) {
	modelUsers, err := gorm.G[model.User](s.DB).Find(ctx)
	if err != nil {
		return nil, err
	}

	simpleUsers := make([]dto.SimpleUser, 0, len(modelUsers))
	for _, mu := range modelUsers {
		simpleUsers = append(simpleUsers, *userToSimpleUser(&mu))
	}

	return simpleUsers, nil
}

func (s *UserService) GetUserFriends(ctx context.Context, userID uint) ([]dto.FriendResponse, error) {
	friends, err := gorm.G[model.Friend](s.DB).Preload("Friend", nil).Where("user_id = ?", userID).Find(ctx)
	if err != nil {
		return nil, err
	}

	onlineStatus, err := s.getOnlineStatus(ctx)
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

	return friendResponses, nil
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
