package testutil

import "testing"

func ClearEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		t.Setenv(key, "")
	}
}
