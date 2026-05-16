package main

import "testing"

func TestFilesSetAppends(t *testing.T) {
	var f files
	if err := f.Set("a.png"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := f.Set("b.jpg"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if len(f) != 2 || f[0] != "a.png" || f[1] != "b.jpg" {
		t.Fatalf("unexpected files: %#v", f)
	}
}

func TestFilesStringEmpty(t *testing.T) {
	var f files
	if got := f.String(); got != "" {
		t.Errorf("empty String() = %q, want \"\"", got)
	}
}

func TestFilesStringJoined(t *testing.T) {
	f := files{"a.png", "b.jpg"}
	if got := f.String(); got != "a.png,b.jpg" {
		t.Errorf("String() = %q, want %q", got, "a.png,b.jpg")
	}
}
