package main

import (
	"strings"
	"testing"
)

func TestFormatTweetsTextEmpty(t *testing.T) {
	if got := formatTweetsText(V2TweetsResponse{}); got != "No tweets found." {
		t.Errorf("got %q, want %q", got, "No tweets found.")
	}
}

func TestFormatTweetsTextSingle(t *testing.T) {
	res := V2TweetsResponse{
		Data: []V2Tweet{{ID: "1", Text: "hi", AuthorID: "u1"}},
		Includes: V2Includes{
			Users: []V2User{{ID: "u1", Name: "Alice", Username: "alice"}},
		},
	}
	got := formatTweetsText(res)
	want := "@alice (Alice) [1]:\nhi"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatTweetsTextOrderReversed(t *testing.T) {
	res := V2TweetsResponse{
		Data: []V2Tweet{
			{ID: "2", Text: "newer", AuthorID: "u1"},
			{ID: "1", Text: "older", AuthorID: "u1"},
		},
		Includes: V2Includes{
			Users: []V2User{{ID: "u1", Name: "Alice", Username: "alice"}},
		},
	}
	got := formatTweetsText(res)
	older := strings.Index(got, "older")
	newer := strings.Index(got, "newer")
	if older < 0 || newer < 0 || older >= newer {
		t.Errorf("expected older before newer, got %q", got)
	}
}

func TestFormatTweetsTextUnescapesHTML(t *testing.T) {
	res := V2TweetsResponse{
		Data: []V2Tweet{{ID: "1", Text: "a &amp; b", AuthorID: "u1"}},
		Includes: V2Includes{
			Users: []V2User{{ID: "u1", Name: "Alice", Username: "alice"}},
		},
	}
	if !strings.Contains(formatTweetsText(res), "a & b") {
		t.Errorf("HTML entity not unescaped: %q", formatTweetsText(res))
	}
}

func TestFormatTweetsTextExpandsRetweet(t *testing.T) {
	res := V2TweetsResponse{
		Data: []V2Tweet{{
			ID: "1", Text: "ignored", AuthorID: "u1",
			ReferencedTweets: []V2ReferencedTweet{{Type: "retweeted", ID: "src"}},
		}},
		Includes: V2Includes{
			Users:  []V2User{{ID: "u1", Username: "alice"}},
			Tweets: []V2Tweet{{ID: "src", Text: "original"}},
		},
	}
	if !strings.Contains(formatTweetsText(res), "RT: original") {
		t.Errorf("retweet not expanded: %q", formatTweetsText(res))
	}
}
