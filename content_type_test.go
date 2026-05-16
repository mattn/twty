package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestContentTypeOfPNG(t *testing.T) {
	header := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	path := writeTempFile(t, "x.png", header)
	got, err := contentTypeOf(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "image/png" {
		t.Errorf("got %q, want %q", got, "image/png")
	}
}

func TestContentTypeOfJPEG(t *testing.T) {
	header := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	path := writeTempFile(t, "x.jpg", header)
	got, err := contentTypeOf(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "image/jpeg" {
		t.Errorf("got %q, want %q", got, "image/jpeg")
	}
}

func TestContentTypeOfGIF(t *testing.T) {
	header := []byte("GIF89a")
	path := writeTempFile(t, "x.gif", header)
	got, err := contentTypeOf(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "image/gif" {
		t.Errorf("got %q, want %q", got, "image/gif")
	}
}

func TestContentTypeOfMissing(t *testing.T) {
	_, err := contentTypeOf(filepath.Join(t.TempDir(), "no-such-file"))
	if err == nil {
		t.Errorf("expected error for missing file")
	}
}

func TestContentTypeOfPlainText(t *testing.T) {
	path := writeTempFile(t, "x.txt", []byte("hello world"))
	got, err := contentTypeOf(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "text/plain") {
		t.Errorf("got %q, want text/plain prefix", got)
	}
}
