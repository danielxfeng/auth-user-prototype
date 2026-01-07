package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/service"
)

type UserHandler struct {
	S *service.UserService
}

// CreateUserHandler godoc
// @Summary Create user
// @Description Register a new user
// @Tags auth/user
// @Accept json
// @Produce json
// @Param body body dto.CreateUserRequest true "Create user payload"
// @Success 201 {object} dto.UserWithoutTokenResponse
// @Router /users/ [post]
func (h *UserHandler) CreateUserHandler(c *gin.Context) {
}

// LoginUserHandler godoc
// @Summary Login user
// @Description Authenticate a user
// @Tags auth/user
// @Accept json
// @Produce json
// @Param body body dto.LoginUserRequest true "Login user payload"
// @Success 200 {object} dto.UserWithTokenResponse
// @Failure 428 {object} dto.TwoFaPendingUserResponse
// @Router /users/loginByIdentifier [post]
func (h *UserHandler) LoginUserHandler(c *gin.Context) {
}

// GetLoggedUserProfileHandler godoc
// @Summary Get current user profile
// @Description Returns the authenticated user's profile
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.UserWithoutTokenResponse
// @Router /users/me [get]
func (h *UserHandler) GetLoggedUserProfileHandler(c *gin.Context) {
}

// UpdateLoggedUserPasswordHandler godoc
// @Summary Update password
// @Description Change password for the authenticated user
// @Tags auth/user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.UpdateUserPasswordRequest true "Update password payload"
// @Success 200 {object} dto.UserWithTokenResponse
// @Router /users/password [post]
func (h *UserHandler) UpdateLoggedUserPasswordHandler(c *gin.Context) {
}

// UpdateLoggedUserProfileHandler godoc
// @Summary Update profile
// @Description Update username or avatar for the authenticated user
// @Tags auth/user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.UpdateUserRequest true "Update profile payload"
// @Success 200 {object} dto.UserWithoutTokenResponse
// @Router /users/me [put]
func (h *UserHandler) UpdateLoggedUserProfileHandler(c *gin.Context) {
}

// DeleteLoggedUserHandler godoc
// @Summary Delete account
// @Description Delete the authenticated user's account
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 204 {object} nil
// @Router /users/me [delete]
func (h *UserHandler) DeleteLoggedUserHandler(c *gin.Context) {
}

// StartTwoFaSetupHandler godoc
// @Summary Start 2FA setup
// @Description Initiate 2FA setup and return setup token and secret
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.TwoFASetupResponse
// @Router /users/2fa/setup [post]
func (h *UserHandler) StartTwoFaSetupHandler(c *gin.Context) {
}

// ConfirmTwoFaSetupHandler godoc
// @Summary Confirm 2FA setup
// @Description Confirm 2FA setup using the provided code and setup token
// @Tags auth/user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.TwoFAConfirmRequest true "2FA confirm payload"
// @Success 200 {object} dto.UserWithTokenResponse
// @Router /users/2fa/confirm [post]
func (h *UserHandler) ConfirmTwoFaSetupHandler(c *gin.Context) {
}

// DisableTwoFaHandler godoc
// @Summary Disable 2FA
// @Description Disable 2FA for the authenticated user
// @Tags auth/user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.DisableTwoFARequest true "Disable 2FA payload"
// @Success 200 {object} dto.UserWithTokenResponse
// @Router /users/2fa/disable [put]
func (h *UserHandler) DisableTwoFaHandler(c *gin.Context) {
}

// TwoFaSubmitHandler godoc
// @Summary Submit 2FA challenge
// @Description Submit 2FA code during login to obtain a user token
// @Tags auth/user
// @Accept json
// @Produce json
// @Param body body dto.TwoFAChallengeRequest true "2FA challenge payload"
// @Success 200 {object} dto.UserWithTokenResponse
// @Router /users/2fa [post]
func (h *UserHandler) TwoFaSubmitHandler(c *gin.Context) {
}

// GetUsersWithLimitedInfoHandler godoc
// @Summary List users (limited info)
// @Description Returns a list of users with limited fields
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.UsersResponse
// @Router /users [get]
func (h *UserHandler) GetUsersWithLimitedInfoHandler(c *gin.Context) {
}

// GetUserLimitedInfoByUsernameHandler godoc
// @Summary Get user by username (limited info)
// @Description Fetch a user's limited info by username
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Param username path string true "Username"
// @Success 200 {object} dto.SimpleUser
// @Router /users/{username} [get]
func (h *UserHandler) GetUserLimitedInfoByUsernameHandler(c *gin.Context) {
}

// GetLoggedUsersFriendsHandler godoc
// @Summary List friends
// @Description Returns the authenticated user's friends
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.FriendsResponse
// @Router /users/friends [get]
func (h *UserHandler) GetLoggedUsersFriendsHandler(c *gin.Context) {
}

// AddFriendHandler godoc
// @Summary Add friend
// @Description Add a new friend for the authenticated user
// @Tags auth/user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.AddNewFriendRequest true "Add friend payload"
// @Success 201 {object} dto.FriendResponse
// @Router /users/friends [post]
func (h *UserHandler) AddFriendHandler(c *gin.Context) {
}

// RemoveFriendHandler godoc
// @Summary Remove friend
// @Description Remove a friend by user id
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Param userId path int true "Friend user id"
// @Success 204 {object} nil
// @Router /users/friends/{userId} [delete]
func (h *UserHandler) RemoveFriendHandler(c *gin.Context) {
}

// ValidateUserHandler godoc
// @Summary Validate user for game service
// @Description Validate user existence and return minimal info
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.UserValidationResponse
// @Router /users/validate [post]
func (h *UserHandler) ValidateUserHandler(c *gin.Context) {
}

// GoogleLoginHandler godoc
// @Summary Google OAuth login
// @Description Start Google OAuth flow and return redirect URL
// @Tags auth/user
// @Produce redirect
// @Success 302 {string} string "Redirect to Google OAuth consent screen"
// @Router /users/google/login [get]
func (h *UserHandler) GoogleLoginHandler(c *gin.Context) {
}

// GoogleCallbackHandler godoc
// @Summary Google OAuth callback
// @Description Handle Google OAuth callback and issue user token
// @Tags auth/user
// @Produce redirect
// @Param code query string true "OAuth code"
// @Param state query string true "OAuth state"
// @Success 302 {string} string "Redirect to frontend with user token"
// @Router /users/google/callback [get]
func (h *UserHandler) GoogleCallbackHandler(c *gin.Context) {
}
