package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	GinMode                         string
	DbAddress                       string
	JwtSecret                       string
	UserTokenExpiry                 int
	OauthStateTokenExpiry           int
	GoogleClientId                  string
	GoogleClientSecret              string
	GoogleRedirectUri               string
	FrontendUrl                     string
	TwoFaUrlPrefix                  string
	TwoFaTokenExpiry                int
	RedisURL                        string
	IsRedisEnabled                  bool
	UserTokenAbsoluteExpiry         int
	Port                            int
	RateLimiterDurationInSec        int
	RateLimiterRequestLimit         int
	RateLimiterCleanupIntervalInSec int
}

func getEnvStrOrDefault(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func getEnvStrOrError(key string) (string, error) {
	value := os.Getenv(key)

	if value == "" {
		return "", fmt.Errorf("environment variable %s is required but not set", key)
	}

	return value, nil
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	strValue := os.Getenv(key)

	intValue, err := strconv.Atoi(strValue)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func LoadConfigFromEnv() (*Config, error) {
	jwtSecret, err := getEnvStrOrError("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	GoogleClientId, err := getEnvStrOrError("GOOGLE_CLIENT_ID")
	if err != nil {
		return nil, err
	}

	GoogleClientSecret, err := getEnvStrOrError("GOOGLE_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}

	return &Config{
		GinMode:                         getEnvStrOrDefault("GIN_MODE", "debug"),
		DbAddress:                       getEnvStrOrDefault("DB_ADDRESS", "data/auth_service_db.sqlite"),
		JwtSecret:                       jwtSecret,
		UserTokenExpiry:                 getEnvIntOrDefault("USER_TOKEN_EXPIRY", 3600),
		OauthStateTokenExpiry:           getEnvIntOrDefault("OAUTH_STATE_TOKEN_EXPIRY", 600),
		GoogleClientId:                  GoogleClientId,
		GoogleClientSecret:              GoogleClientSecret,
		GoogleRedirectUri:               getEnvStrOrDefault("GOOGLE_REDIRECT_URI", "test-google-redirect-uri"),
		FrontendUrl:                     getEnvStrOrDefault("FRONTEND_URL", "http://localhost:5173"),
		TwoFaUrlPrefix:                  getEnvStrOrDefault("TWO_FA_URL_PREFIX", "otpauth://totp/Transcendence?secret="),
		TwoFaTokenExpiry:                getEnvIntOrDefault("TWO_FA_TOKEN_EXPIRY", 600),
		RedisURL:                        getEnvStrOrDefault("REDIS_URL", ""),
		IsRedisEnabled:                  getEnvStrOrDefault("REDIS_URL", "") != "",
		UserTokenAbsoluteExpiry:         getEnvIntOrDefault("USER_TOKEN_ABSOLUTE_EXPIRY", 2592000),
		Port:                            getEnvIntOrDefault("PORT", 3003),
		RateLimiterDurationInSec:        getEnvIntOrDefault("RATE_LIMITER_DURATION_IN_SECONDS", 60),
		RateLimiterRequestLimit:         getEnvIntOrDefault("RATE_LIMITER_REQUEST_LIMIT", 1000),
		RateLimiterCleanupIntervalInSec: getEnvIntOrDefault("RATE_LIMITER_CLEANUP_INTERVAL_IN_SECONDS", 300),
	}, nil
}
