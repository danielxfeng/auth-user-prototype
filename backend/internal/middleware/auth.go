package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/dependency"
	"github.com/paularynty/transcendence/auth-service-go/internal/util/jwt"
)

const PrefixBearer = "Bearer "

func Auth(dep *dependency.Dependency) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, PrefixBearer) {
			_ = c.AbortWithError(401, NewAuthError(401, "Invalid or expired token"))
			return
		}

		tokenString := authHeader[len(PrefixBearer):]

		userJwtPayload, err := jwt.ValidateUserTokenGeneric(dep, tokenString)
		if err != nil {
			_ = c.AbortWithError(401, NewAuthError(401, "Invalid or expired token"))
			return
		}

		c.Set("userID", userJwtPayload.UserID)
		c.Set("token", tokenString)

		c.Next()
	}
}
