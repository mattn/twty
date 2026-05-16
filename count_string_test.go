package main

import "testing"

func TestCountStringPositive(t *testing.T) {
	if got := countString(10); got != "10" {
		t.Errorf("countString(10) = %q, want %q", got, "10")
	}
}

func TestCountStringZero(t *testing.T) {
	if got := countString(0); got != "" {
		t.Errorf("countString(0) = %q, want \"\"", got)
	}
}

func TestCountStringNegative(t *testing.T) {
	if got := countString(-5); got != "" {
		t.Errorf("countString(-5) = %q, want \"\"", got)
	}
}
