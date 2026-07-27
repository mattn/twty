package main

import (
	"testing"
)

func TestParseTweetTarget(t *testing.T) {
	tests := []struct {
		input    string
		tweetID  string
		username string
		wantErr  bool
	}{
		{"1234567890123456789", "1234567890123456789", "", false},
		{"https://x.com/mattn_jp/status/1234567890123456789", "1234567890123456789", "", false},
		{"https://twitter.com/mattn_jp/status/1234567890123456789", "1234567890123456789", "", false},
		{"https://mobile.twitter.com/mattn_jp/status/1234567890123456789", "1234567890123456789", "", false},
		{"https://x.com/mattn_jp/status/1234567890123456789?s=20&t=abc", "1234567890123456789", "", false},
		{"https://x.com/mattn_jp/status/1234567890123456789/photo/1", "1234567890123456789", "", false},
		{"https://x.com/i/web/status/1234567890123456789", "1234567890123456789", "", false},
		{"https://twitter.com/statuses/1234567890123456789", "1234567890123456789", "", false},
		{"x.com/mattn_jp/status/1234567890123456789", "1234567890123456789", "", false},
		{"https://x.com/mattn_jp", "", "mattn_jp", false},
		{"https://x.com/@mattn_jp", "", "mattn_jp", false},
		{"x.com/mattn_jp", "", "mattn_jp", false},
		{"https://x.com/", "", "", true},
		{"https://x.com/home", "", "", true},
		{"https://x.com/search?q=golang", "", "", true},
		{"https://x.com/mattn_jp/status/abc", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		tweetID, username, err := parseTweetTarget(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseTweetTarget(%q) expected error, got tweetID=%q username=%q", tt.input, tweetID, username)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseTweetTarget(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if tweetID != tt.tweetID || username != tt.username {
			t.Errorf("parseTweetTarget(%q) = (%q, %q), want (%q, %q)", tt.input, tweetID, username, tt.tweetID, tt.username)
		}
	}
}
