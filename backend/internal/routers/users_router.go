package routers

import (
	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
	"github.com/paularynty/transcendence/auth-service-go/internal/handler"
	"github.com/paularynty/transcendence/auth-service-go/internal/middleware"
	"github.com/paularynty/transcendence/auth-service-go/internal/service"
)

func UsersRouter(r *gin.RouterGroup) {
	h := &handler.UserHandler{Service: service.NewUserService(db.DB)}

	// Public endpoints
	r.POST("/", middleware.ValidateBody[dto.CreateUserRequest](), h.CreateUserHandler)
	r.POST("/loginByIdentifier", middleware.ValidateBody[dto.LoginUserRequest](), h.LoginUserHandler)
	r.POST("/2fa", middleware.ValidateBody[dto.TwoFAChallengeRequest](), h.TwoFaSubmitHandler)
	r.GET("/google/login", h.GoogleLoginHandler)
	r.GET("/google/callback", h.GoogleCallbackHandler)

	// Authenticated endpoints
	auth := r.Group("")
	auth.Use(middleware.Auth())

	auth.GET("/me", h.GetLoggedUserProfileHandler)
	auth.PUT("/password", middleware.ValidateBody[dto.UpdateUserPasswordRequest](), h.UpdateLoggedUserPasswordHandler)
	auth.PUT("/me", middleware.ValidateBody[dto.UpdateUserRequest](), h.UpdateLoggedUserProfileHandler)
	auth.DELETE("/logout", h.LogoutUserHandler)
	auth.DELETE("/me", h.DeleteLoggedUserHandler)

	auth.POST("/2fa/setup", h.StartTwoFaSetupHandler)
	auth.POST("/2fa/confirm", middleware.ValidateBody[dto.TwoFAConfirmRequest](), h.ConfirmTwoFaSetupHandler)
	auth.PUT("/2fa/disable", middleware.ValidateBody[dto.DisableTwoFARequest](), h.DisableTwoFaHandler)

	auth.GET("/friends", h.GetLoggedUsersFriendsHandler)
	auth.POST("/friends", middleware.ValidateBody[dto.AddNewFriendRequest](), h.AddFriendHandler)

	auth.POST("/validate", h.ValidateUserHandler)
	auth.GET("/", h.GetUsersWithLimitedInfoHandler)
}
