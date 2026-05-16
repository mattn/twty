package main

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateCodeVerifierLength(t *testing.T) {
	v, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 32 random bytes → base64 raw url encoding → 43 chars
	if len(v) != 43 {
		t.Errorf("len(verifier) = %d, want 43", len(v))
	}
}

func TestGenerateCodeVerifierURLSafe(t *testing.T) {
	v, err := generateCodeVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.ContainsAny(v, "+/=") {
		t.Errorf("verifier %q contains non URL-safe chars", v)
	}
}

func TestGenerateCodeVerifierUnique(t *testing.T) {
	a, _ := generateCodeVerifier()
	b, _ := generateCodeVerifier()
	if a == b {
		t.Errorf("two verifiers collided: %q", a)
	}
}

func TestGenerateCodeChallengeMatchesSHA256(t *testing.T) {
	verifier := "test-verifier"
	got := generateCodeChallenge(verifier)
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGenerateStateLength(t *testing.T) {
	s, err := generateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 16 random bytes → base64 raw url encoding → 22 chars
	if len(s) != 22 {
		t.Errorf("len(state) = %d, want 22", len(s))
	}
}

func TestGenerateStateUnique(t *testing.T) {
	a, _ := generateState()
	b, _ := generateState()
	if a == b {
		t.Errorf("two states collided: %q", a)
	}
}
