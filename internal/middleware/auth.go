package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/paularynty/transcendence/auth-service-go/internal/util"
)

const PrefixBearer = "Bearer "

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" || len(authHeader) < len(PrefixBearer) || authHeader[:len(PrefixBearer)] != PrefixBearer {
			c.AbortWithError(401, NewAuthError(401, "Invalid or expired token"))
			return
		}

		tokenString := authHeader[len(PrefixBearer):]

		userJwtPayload, err := util.ValidateUserTokenGeneric(tokenString)
		if err != nil {
			c.AbortWithError(401, NewAuthError(401, "Invalid or expired token"))
			return
		}

		c.Set("userID", userJwtPayload.UserID)

		c.Next()
	}
}
