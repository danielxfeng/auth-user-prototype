package config

import (
	"os"
	"strconv"
)

type Config struct {
	GinMode               string
	DbAddress             string
	JwtSecret             string
	UserTokenExpiry       int
	OauthStateTokenExpiry int
	GoogleClientId        string
	GoogleClientSecret    string
	GoogleRedirectUri     string
	FrontendUrl           string
	TwoFaUrlPrefix        string
	TwoFaTokenExpiry      int
}

var Cfg *Config

func getEnvStrOrDefault(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	strValue := os.Getenv(key)

	intValue, err := strconv.Atoi(strValue)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func LoadConfig() {
	Cfg = &Config{
		GinMode:               getEnvStrOrDefault("GIN_MODE", "debug"),
		DbAddress:             getEnvStrOrDefault("DB_ADDRESS", "data/auth_service_db.sqlite"),
		JwtSecret:             getEnvStrOrDefault("JWT_SECRET", "test-secret"),
		UserTokenExpiry:       getEnvIntOrDefault("USER_TOKEN_EXPIRY", 3600),
		OauthStateTokenExpiry: getEnvIntOrDefault("OAUTH_STATE_TOKEN_EXPIRY", 600),
		GoogleClientId:        getEnvStrOrDefault("GOOGLE_CLIENT_ID", "test-google-client-id"),
		GoogleClientSecret:    getEnvStrOrDefault("GOOGLE_CLIENT_SECRET", "test-google-client-secret"),
		GoogleRedirectUri:     getEnvStrOrDefault("GOOGLE_REDIRECT_URI", "test-google-redirect-uri"),
		FrontendUrl:           getEnvStrOrDefault("FRONTEND_URL", "http://localhost:5173"),
		TwoFaUrlPrefix:        getEnvStrOrDefault("TWO_FA_URL_PREFIX", "otpauth://totp/Transcendence?secret="),
		TwoFaTokenExpiry:      getEnvIntOrDefault("TWO_FA_TOKEN_EXPIRY", 600),
	}
}
