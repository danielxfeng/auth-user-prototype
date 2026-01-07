package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
)

func ValidateBody[T any]() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body T
		if err := c.ShouldBindJSON(&body); err != nil {
			c.AbortWithError(400, NewAuthError(400, err.Error()))
			return
		}

		if err := dto.Validate.Struct(&body); err != nil {
			c.AbortWithError(400, err)
			return
		}

		c.Set("validatedBody", body)

		c.Next()
	}
}