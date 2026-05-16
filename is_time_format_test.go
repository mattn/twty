package main

import "testing"

func TestIsTimeFormat(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"iso date", "2024-05-01", true},
		{"single digits", "1-2-3", true},
		{"two parts", "2024-05", false},
		{"four parts", "2024-05-01-02", false},
		{"alpha month", "2024-may-01", false},
		{"empty", "", false},
		{"only dashes", "--", false},
		{"trailing dash", "2024-05-", false},
		{"negative", "-2024-05-01", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isTimeFormat(c.in); got != c.want {
				t.Errorf("isTimeFormat(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
