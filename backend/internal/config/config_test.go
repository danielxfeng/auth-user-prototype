package config

import (
	"testing"
)

func assertError(t *testing.T, err error, name string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error for %s", name)
	}
}

func assertNoError(t *testing.T, err error, name string) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error for %s: %v", name, err)
	}
}

func TestGetEnvStrOrDefault(t *testing.T) {
	t.Setenv("TEST_STR", "")
	if got := getEnvStrOrDefault("TEST_STR", "fallback"); got != "fallback" {
		t.Fatalf("expected default value, got %q", got)
	}

	t.Setenv("TEST_STR", "value")
	if got := getEnvStrOrDefault("TEST_STR", "fallback"); got != "value" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestGetEnvStrOrError(t *testing.T) {
	t.Setenv("TEST_PANIC", "")
	_, err := getEnvStrOrError("TEST_PANIC")
	assertError(t, err, "empty env")

	t.Setenv("TEST_PANIC", "value")
	got, err := getEnvStrOrError("TEST_PANIC")
	assertNoError(t, err, "set env")
	if got != "value" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestGetEnvIntOrDefault(t *testing.T) {
	t.Setenv("TEST_INT", "")
	if got := getEnvIntOrDefault("TEST_INT", 7); got != 7 {
		t.Fatalf("expected default value, got %d", got)
	}

	t.Setenv("TEST_INT", "42")
	if got := getEnvIntOrDefault("TEST_INT", 7); got != 42 {
		t.Fatalf("expected env value, got %d", got)
	}

	t.Setenv("TEST_INT", "not-an-int")
	if got := getEnvIntOrDefault("TEST_INT", 7); got != 7 {
		t.Fatalf("expected default value for invalid int, got %d", got)
	}
}

func TestLoadConfigFromEnv_ErrsOnMissingRequired(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("GOOGLE_CLIENT_ID", "client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret")

	_, err := LoadConfigFromEnv()
	assertNoError(t, err, "all required set")

	t.Setenv("JWT_SECRET", "")
	_, err = LoadConfigFromEnv()
	assertError(t, err, "JWT_SECRET unset")

	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	_, err = LoadConfigFromEnv()
	assertError(t, err, "GOOGLE_CLIENT_ID unset")

	t.Setenv("GOOGLE_CLIENT_ID", "client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	_, err = LoadConfigFromEnv()
	assertError(t, err, "GOOGLE_CLIENT_SECRET unset")
}
