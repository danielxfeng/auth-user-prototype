package authError

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
