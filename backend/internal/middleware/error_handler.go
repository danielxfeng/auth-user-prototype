package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthError struct {
	Status  int
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

func NewAuthError(status int, message string) *AuthError {
	return &AuthError{
		Status:  status,
		Message: message,
	}
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		errs := c.Errors

		if len(errs) == 0 {
			return
		}

		err := c.Errors.Last().Err

		var authErr *AuthError

		// Handle AuthError specifically
		if errors.As(err, &authErr) {
			c.AbortWithStatusJSON(authErr.Status, gin.H{
				"error": authErr.Message,
			})
			return
		}

		var validationErr validator.ValidationErrors

		// Handle validation errors
		if errors.As(err, &validationErr) {
			messages := make([]string, 0, len(validationErr))
			for _, fe := range validationErr {
				messages = append(messages, fe.Error())
			}
			c.AbortWithStatusJSON(400, gin.H{
				"error": messages,
			})
			return
		}

		// Handle other error types or default to 500
		c.AbortWithStatusJSON(500, gin.H{
			"error": "Internal Server Error",
		})
	}
}

func PanicHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		c.AbortWithStatusJSON(500, gin.H{
			"error": "Internal Server Error",
		})
	})
}
