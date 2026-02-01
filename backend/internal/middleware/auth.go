package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	authError "github.com/paularynty/transcendence/auth-service-go/internal/auth_error"
	"github.com/paularynty/transcendence/auth-service-go/internal/service"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

const PrefixBearer = "Bearer "

func Auth(userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, PrefixBearer) {
			_ = c.AbortWithError(401, authError.NewAuthError(401, "Invalid or expired token"))
			return
		}

		tokenString := authHeader[len(PrefixBearer):]

		userJwtPayload, err := jwt.ValidateUserTokenGeneric(userService.Dep, tokenString)
		if err != nil {
			_ = c.AbortWithError(401, authError.NewAuthError(401, "Invalid or expired token"))
			return
		}

		err = userService.ValidateUserToken(c.Request.Context(), tokenString, userJwtPayload.UserID)
		
		var authError *authError.AuthError
		if err != nil {
			if errors.As(err, &authError) && authError.Status == 401 {
				_ = c.AbortWithError(401, authError)
				return
			}
			_ = c.AbortWithError(500, err)
			return
		}

		c.Set("userID", userJwtPayload.UserID)
		c.Set("token", tokenString)

		c.Next()
	}
}
