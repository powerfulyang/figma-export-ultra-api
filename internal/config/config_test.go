package config

import (
	"os"
	"testing"
)

func TestGetIntBool(t *testing.T) {
	os.Setenv("X_INT", "42")
	t.Cleanup(func() { os.Unsetenv("X_INT") })
	if v := getInt("X_INT", 1); v != 42 {
		t.Fatalf("want 42, got %d", v)
	}

	os.Setenv("X_BOOL_T", "true")
	os.Setenv("X_BOOL_F", "false")
	t.Cleanup(func() { os.Unsetenv("X_BOOL_T"); os.Unsetenv("X_BOOL_F") })
	if !getBool("X_BOOL_T", false) {
		t.Fatalf("want true")
	}
	if getBool("X_BOOL_F", true) {
		t.Fatalf("want false")
	}
}
