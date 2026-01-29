package config

import (
	"testing"
)

func assertPanics(t *testing.T, fn func(), name string) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for %s", name)
		}
	}()
	fn()
}

func assertNotPanics(t *testing.T, fn func(), name string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic for %s: %v", name, r)
		}
	}()
	fn()
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

func TestGetEnvStrOrPanic(t *testing.T) {
	t.Setenv("TEST_PANIC", "")
	assertPanics(t, func() {
		_ = getEnvStrOrPanic("TEST_PANIC")
	}, "empty env")

	t.Setenv("TEST_PANIC", "value")
	assertNotPanics(t, func() {
		if got := getEnvStrOrPanic("TEST_PANIC"); got != "value" {
			t.Fatalf("expected env value, got %q", got)
		}
	}, "set env")
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

func TestLoadConfigFromEnv_PanicsOnMissingRequired(t *testing.T) {
	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("GOOGLE_CLIENT_ID", "client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret")

	assertNotPanics(t, func() {
		_ = LoadConfigFromEnv()
	}, "all required set")

	t.Setenv("JWT_SECRET", "")
	assertPanics(t, func() {
		_ = LoadConfigFromEnv()
	}, "JWT_SECRET unset")

	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	assertPanics(t, func() {
		_ = LoadConfigFromEnv()
	}, "GOOGLE_CLIENT_ID unset")

	t.Setenv("GOOGLE_CLIENT_ID", "client")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	assertPanics(t, func() {
		_ = LoadConfigFromEnv()
	}, "GOOGLE_CLIENT_SECRET unset")
}
