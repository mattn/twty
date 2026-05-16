package main

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigDirCreatesPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	dir, err := configDir()
	if err != nil {
		t.Fatalf("configDir error: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}
	if filepath.Base(dir) != "twty" {
		t.Errorf("expected basename twty, got %q", dir)
	}
}

func TestConfigDirUsesHomeOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only path")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir, err := configDir()
	if err != nil {
		t.Fatalf("configDir error: %v", err)
	}
	want := filepath.Join(home, ".config", "twty")
	if dir != want {
		t.Errorf("got %q, want %q", dir, want)
	}
}
