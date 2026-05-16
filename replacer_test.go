package main

import "testing"

func TestReplacerStripsCR(t *testing.T) {
	if got := replacer.Replace("a\rb"); got != "ab" {
		t.Errorf("got %q, want %q", got, "ab")
	}
}

func TestReplacerLFToSpace(t *testing.T) {
	if got := replacer.Replace("a\nb"); got != "a b" {
		t.Errorf("got %q, want %q", got, "a b")
	}
}

func TestReplacerTabToSpace(t *testing.T) {
	if got := replacer.Replace("a\tb"); got != "a b" {
		t.Errorf("got %q, want %q", got, "a b")
	}
}

func TestReplacerMixed(t *testing.T) {
	in := "a\r\nb\tc"
	if got := replacer.Replace(in); got != "a b c" {
		t.Errorf("got %q, want %q", got, "a b c")
	}
}
