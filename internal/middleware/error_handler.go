package middleware

import "github.com/gin-gonic/gin"

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

		// Handle AuthError specifically
		if authErr, ok := err.(*AuthError); ok {
			c.JSON(authErr.Status, gin.H{
				"error": authErr.Message,
			})
			return
		}

		// Handle other error types or default to 500
		c.JSON(500, gin.H{
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
