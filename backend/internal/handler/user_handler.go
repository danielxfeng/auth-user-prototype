package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/service"
)

type UserHandler struct {
	Service *service.UserService
}

func handleError(c *gin.Context, err error) {
	var authErr *middleware.AuthError
	if errors.As(err, &authErr) {
		_ = c.AbortWithError(authErr.Status, err)
	} else {
		_ = c.AbortWithError(500, err)
	}
}

func (h *UserHandler) validateToken(c *gin.Context) (uint, error) {
	userID := c.MustGet("userID").(uint)
	token := c.MustGet("token").(string)

	err := h.Service.ValidateUserToken(c.Request.Context(), token, userID)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

// @BasePath /users

// CreateUserHandler godoc
// @Summary Create user
// @Description Register a new user
// @Tags auth/user
// @Accept json
// @Produce json
// @Param body body dto.CreateUserRequest true "Create user payload"
// @Success 201 {object} dto.UserWithoutTokenResponse
// @Router / [post]
func (h *UserHandler) CreateUserHandler(c *gin.Context) {
	request := c.MustGet("validatedBody").(dto.CreateUserRequest)

	user, e := h.Service.CreateUser(c.Request.Context(), &request)
	if e != nil {
		handleError(c, e)
		return
	}

	c.JSON(201, user)
}

// LoginUserHandler godoc
// @Summary Login user
// @Description Authenticate a user
// @Tags auth/user
// @Accept json
// @Produce json
// @Param body body dto.LoginUserRequest true "Login user payload"
// @Success 200 {object} dto.UserWithTokenResponse
// @Failure 428 {object} dto.TwoFAPendingUserResponse
// @Router /loginByIdentifier [post]
func (h *UserHandler) LoginUserHandler(c *gin.Context) {
	request := c.MustGet("validatedBody").(dto.LoginUserRequest)

	user, e := h.Service.LoginUser(c.Request.Context(), &request)
	if e != nil {
		handleError(c, e)
		return
	}

	if user.TwoFAPending != nil {
		c.JSON(428, user.TwoFAPending)
		c.Abort()
		return
	}

	if user.User == nil {
		handleError(c, errors.New("missing user payload"))
		return
	}

	c.JSON(200, user.User)
}

// LogoutUserHandler godoc
// @Summary Logout user
// @Description Logout the authenticated user
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 204 {object} nil
// @Router /logout [delete]
func (h *UserHandler) LogoutUserHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	err = h.Service.LogoutUser(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.Status(204)
}

// GetLoggedUserProfileHandler godoc
// @Summary Get current user profile
// @Description Returns the authenticated user's profile
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.UserWithoutTokenResponse
// @Router /me [get]
func (h *UserHandler) GetLoggedUserProfileHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	user, err := h.Service.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, user)
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
// @Router /password [put]
func (h *UserHandler) UpdateLoggedUserPasswordHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	request := c.MustGet("validatedBody").(dto.UpdateUserPasswordRequest)

	user, err := h.Service.UpdateUserPassword(c.Request.Context(), userID, &request)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, user)
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
// @Router /me [put]
func (h *UserHandler) UpdateLoggedUserProfileHandler(c *gin.Context) {
	userId, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	request := c.MustGet("validatedBody").(dto.UpdateUserRequest)

	user, err := h.Service.UpdateUserProfile(c.Request.Context(), userId, &request)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, user)
}

// DeleteLoggedUserHandler godoc
// @Summary Delete account
// @Description Delete the authenticated user's account
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 204 {object} nil
// @Router /me [delete]
func (h *UserHandler) DeleteLoggedUserHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	err = h.Service.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.Status(204)
}

// StartTwoFaSetupHandler godoc
// @Summary Start 2FA setup
// @Description Initiate 2FA setup and return setup token and secret
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.TwoFASetupResponse
// @Router /2fa/setup [post]
func (h *UserHandler) StartTwoFaSetupHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	response, err := h.Service.StartTwoFaSetup(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, response)
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
// @Router /2fa/confirm [post]
func (h *UserHandler) ConfirmTwoFaSetupHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	request := c.MustGet("validatedBody").(dto.TwoFAConfirmRequest)

	user, err := h.Service.ConfirmTwoFaSetup(c.Request.Context(), userID, &request)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, user)
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
// @Router /2fa/disable [put]
func (h *UserHandler) DisableTwoFaHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	request := c.MustGet("validatedBody").(dto.DisableTwoFARequest)

	user, err := h.Service.DisableTwoFA(c.Request.Context(), userID, &request)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, user)
}

// TwoFaSubmitHandler godoc
// @Summary Submit 2FA challenge
// @Description Submit 2FA code during login to obtain a user token
// @Tags auth/user
// @Accept json
// @Produce json
// @Param body body dto.TwoFAChallengeRequest true "2FA challenge payload"
// @Success 200 {object} dto.UserWithTokenResponse
// @Router /2fa [post]
func (h *UserHandler) TwoFaSubmitHandler(c *gin.Context) {
	request := c.MustGet("validatedBody").(dto.TwoFAChallengeRequest)

	user, err := h.Service.SubmitTwoFAChallenge(c.Request.Context(), &request)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, user)
}

// GetUsersWithLimitedInfoHandler godoc
// @Summary List users (limited info)
// @Description Returns a list of users with limited fields
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {array} dto.SimpleUser
// @Router / [get]
func (h *UserHandler) GetUsersWithLimitedInfoHandler(c *gin.Context) {
	_, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	users, err := h.Service.GetAllUsersLimitedInfo(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, users)
}

// GetLoggedUsersFriendsHandler godoc
// @Summary List friends
// @Description Returns the authenticated user's friends
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {array} dto.FriendResponse
// @Router /friends [get]
func (h *UserHandler) GetLoggedUsersFriendsHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	friends, err := h.Service.GetUserFriends(c.Request.Context(), userID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, friends)
}

// AddFriendHandler godoc
// @Summary Add friend
// @Description Add a new friend for the authenticated user
// @Tags auth/user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body dto.AddNewFriendRequest true "Add friend payload"
// @Success 201 {object} nil
// @Router /friends [post]
func (h *UserHandler) AddFriendHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	request := c.MustGet("validatedBody").(dto.AddNewFriendRequest)

	err = h.Service.AddNewFriend(c.Request.Context(), userID, &request)
	if err != nil {
		handleError(c, err)
		return
	}

	c.Status(201)
}

// ValidateUserHandler godoc
// @Summary Validate user for game service
// @Description Validate user existence and return minimal info
// @Tags auth/user
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.UserValidationResponse
// @Router /validate [post]
func (h *UserHandler) ValidateUserHandler(c *gin.Context) {
	userID, err := h.validateToken(c)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(200, dto.UserValidationResponse{UserID: userID})
}

// GoogleLoginHandler godoc
// @Summary Google OAuth login
// @Description Start Google OAuth flow and return redirect URL
// @Tags auth/user
// @Success 302 {string} string "Redirect to Google OAuth consent screen"
// @Router /google/login [get]
func (h *UserHandler) GoogleLoginHandler(c *gin.Context) {
	url, err := h.Service.GetGoogleOAuthURL(c.Request.Context())
	if err != nil {
		handleError(c, err)
		return
	}

	c.Redirect(302, url)
}

// GoogleCallbackHandler godoc
// @Summary Google OAuth callback
// @Description Handle Google OAuth callback and issue user token
// @Tags auth/user
// @Param code query string true "OAuth code"
// @Param state query string true "OAuth state"
// @Success 302 {string} string "Redirect to frontend with user token"
// @Router /google/callback [get]
func (h *UserHandler) GoogleCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		handleError(c, middleware.NewAuthError(400, "Missing code or state in callback"))
		return
	}

	url := h.Service.HandleGoogleOAuthCallback(c.Request.Context(), code, state)

	if url == "" {
		handleError(c, middleware.NewAuthError(500, "Failed to process Google OAuth callback"))
		return
	}

	c.Redirect(302, url)
}
