package main

import (
	"os"
	"testing"
	"time"
)

func TestDefaultTTLFromEnv(t *testing.T) {
	os.Setenv("PTOR_TTL_SECONDS", "10")
	defer os.Unsetenv("PTOR_TTL_SECONDS")
	if got := defaultTTL(); got != 10*time.Second {
		t.Fatalf("expected 10s, got %v", got)
	}
}
