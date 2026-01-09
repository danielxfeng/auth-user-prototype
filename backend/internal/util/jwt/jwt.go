package jwt

import (
	"time"

	libjwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/dto"
)

const (
	UserTokenType        = "USER"
	GoogleOAuthStateType = "GoogleOAuthState"
	TwoFASetupType       = "2FA_SETUP"
	TwoFATokenType       = "2FA"
)

func generateRegisteredClaims(expiration int) libjwt.RegisteredClaims {
	return libjwt.RegisteredClaims{
		ExpiresAt: libjwt.NewNumericDate(time.Now().Add(time.Duration(expiration) * time.Second)),
		IssuedAt:  libjwt.NewNumericDate(time.Now()),
		ID:        uuid.New().String(),
	}
}

func SignUserToken(userID uint) (string, error) {
	claims := dto.UserJwtPayload{
		UserID:           userID,
		Type:             UserTokenType,
		RegisteredClaims: generateRegisteredClaims(config.Cfg.UserTokenExpiry),
	}

	token := libjwt.NewWithClaims(libjwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.Cfg.JwtSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func SignOauthStateToken() (string, error) {
	claims := dto.OauthStateJwtPayload{
		Type:             GoogleOAuthStateType,
		RegisteredClaims: generateRegisteredClaims(config.Cfg.OauthStateTokenExpiry),
	}

	token := libjwt.NewWithClaims(libjwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.Cfg.JwtSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func SignTwoFASetupToken(userID uint, secret string) (string, error) {
	claims := dto.TwoFaSetupJwtPayload{
		UserID:           userID,
		Secret:           secret,
		Type:             TwoFASetupType,
		RegisteredClaims: generateRegisteredClaims(config.Cfg.TwoFaTokenExpiry),
	}

	token := libjwt.NewWithClaims(libjwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.Cfg.JwtSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func SignTwoFAToken(userID uint) (string, error) {
	claims := dto.TwoFaJwtPayload{
		UserID:           userID,
		Type:             TwoFATokenType,
		RegisteredClaims: generateRegisteredClaims(config.Cfg.TwoFaTokenExpiry),
	}

	token := libjwt.NewWithClaims(libjwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(config.Cfg.JwtSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func validateToken[T libjwt.Claims](signedToken string, claims T) (T, error) {
	token, err := libjwt.ParseWithClaims(
		signedToken,
		claims,
		func(token *libjwt.Token) (any, error) {
			return []byte(config.Cfg.JwtSecret), nil
		},
	)
	if err != nil {
		return claims, err
	}

	if !token.Valid {
		return claims, libjwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

func ValidateUserTokenGeneric(signedToken string) (*dto.UserJwtPayload, error) {
	claims := &dto.UserJwtPayload{}
	parsedClaims, err := validateToken(signedToken, claims)
	if err != nil {
		return nil, err
	}

	if parsedClaims.Type != UserTokenType {
		return nil, libjwt.ErrTokenInvalidClaims
	}

	return parsedClaims, nil
}

func ValidateOauthStateToken(signedToken string) (*dto.OauthStateJwtPayload, error) {
	claims := &dto.OauthStateJwtPayload{}
	parsedClaims, err := validateToken(signedToken, claims)
	if err != nil {
		return nil, err
	}

	if parsedClaims.Type != GoogleOAuthStateType {
		return nil, libjwt.ErrTokenInvalidClaims
	}

	return parsedClaims, nil
}

func ValidateTwoFAToken(signedToken string) (*dto.TwoFaJwtPayload, error) {
	claims := &dto.TwoFaJwtPayload{}
	parsedClaims, err := validateToken(signedToken, claims)
	if err != nil {
		return nil, err
	}

	if parsedClaims.Type != TwoFATokenType {
		return nil, libjwt.ErrTokenInvalidClaims
	}

	return parsedClaims, nil
}

func ValidateTwoFASetupToken(signedToken string) (*dto.TwoFaSetupJwtPayload, error) {
	claims := &dto.TwoFaSetupJwtPayload{}
	parsedClaims, err := validateToken(signedToken, claims)
	if err != nil {
		return nil, err
	}

	if parsedClaims.Type != TwoFASetupType {
		return nil, libjwt.ErrTokenInvalidClaims
	}

	return parsedClaims, nil
}
