package config

import (
	"fmt"
	"testing"
)

const testKey = "TEST_KEY"
const notSet = "notSet"

func setEnv(t *testing.T, v string) {
	t.Helper()

	if v == notSet {
		return
	}

	t.Setenv(testKey, v)
}

func TestGetEnvStrOrDefault(t *testing.T) {
	const validValue = "v1"
	const defaultValue = "v"
	const emptyValue = ""

	testCases := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "valid env string",
			envValue: validValue,
			expected: validValue,
		},
		{
			name:     "empty env string",
			envValue: emptyValue,
			expected: defaultValue,
		},
		{
			name:     "env not set",
			envValue: notSet,
			expected: defaultValue,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.envValue)

			if got := getEnvStrOrDefault(testKey, defaultValue); got != tc.expected {
				t.Fatalf("expect: %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestGetEnvIntOrDefault(t *testing.T) {
	const validValue = "10"
	const validExpected = 10
	const defaultValue = 22
	const invalidValue = "a"

	testCases := []struct {
		name     string
		envValue string
		expected int
	}{
		{
			name:     "valid env (int)",
			envValue: validValue,
			expected: validExpected,
		},
		{
			name:     "env not set (int)",
			envValue: notSet,
			expected: defaultValue,
		},
		{
			name:     "invalid env (int)",
			envValue: invalidValue,
			expected: defaultValue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.envValue)

			if got := getEnvIntOrDefault(testKey, defaultValue); got != tc.expected {
				t.Fatalf("expected: %d, got: %d", tc.expected, got)
			}
		})
	}
}

func TestGetEnvStrOrError(t *testing.T) {
	const validValue = "v1"
	const emptyValue = ""
	const errorValue = "error"

	testCases := []struct {
		name      string
		envValue  string
		expected  string
		expectErr bool
	}{
		{
			name:      "valid env string",
			envValue:  validValue,
			expected:  validValue,
			expectErr: false,
		},
		{
			name:      "empty env string",
			envValue:  emptyValue,
			expected:  errorValue,
			expectErr: true,
		},
		{
			name:      "env not set",
			envValue:  notSet,
			expected:  errorValue,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.envValue)

			got, err := getEnvStrOrError(testKey)

			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got %q", got)
			}

			if !tc.expectErr && err != nil {
				t.Fatalf("expected %q, got error %v", tc.expected, err)
			}

			if !tc.expectErr && got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

var mandatoryItems = []string{
	"JWT_SECRET",
	"GOOGLE_CLIENT_ID",
	"GOOGLE_CLIENT_SECRET",
}

func setEnvForMandatoryItem(t *testing.T, keys []string) {
	t.Helper()

	for _, key := range mandatoryItems {
		t.Setenv(key, "")
	}

	for _, key := range keys {
		t.Setenv(key, "test_value")
	}
}

func TestLoadConfigFromEnv_MissingMandatory(t *testing.T) {
	type testCase struct {
		name      string
		expectErr bool
		keys      []string
	}

	testCases := []testCase{
		{
			name:      "normal case",
			expectErr: false,
			keys:      mandatoryItems,
		},
	}

	for i, item := range mandatoryItems {
		keys := make([]string, 0, len(mandatoryItems)-1)
		keys = append(keys, mandatoryItems[:i]...)
		keys = append(keys, mandatoryItems[i+1:]...)

		tc := testCase{
			name:      fmt.Sprintf("missing %s", item),
			expectErr: true,
			keys:      keys,
		}
		testCases = append(testCases, tc)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			setEnvForMandatoryItem(t, tc.keys)
			cfg, err := LoadConfigFromEnv()

			if tc.expectErr && err == nil {
				t.Fatalf("expected error, but got cfg: %v.", cfg)
			}

			if !tc.expectErr && cfg == nil {
				t.Fatalf("expected cfg, but got nil")
			}

			if !tc.expectErr && err != nil {
				t.Fatalf("expected cfg, but got err: %v", err)
			}
		})
	}
}
