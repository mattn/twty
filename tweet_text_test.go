package main

import "testing"

func TestTweetTextPlain(t *testing.T) {
	tw := V2Tweet{Text: "hello"}
	if got := tweetText(tw, nil); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestTweetTextRetweet(t *testing.T) {
	tw := V2Tweet{
		Text:             "ignored",
		ReferencedTweets: []V2ReferencedTweet{{Type: "retweeted", ID: "1"}},
	}
	tm := map[string]V2Tweet{"1": {ID: "1", Text: "original"}}
	if got := tweetText(tw, tm); got != "RT: original" {
		t.Errorf("got %q, want %q", got, "RT: original")
	}
}

func TestTweetTextRetweetMissingReference(t *testing.T) {
	tw := V2Tweet{
		Text:             "fallback",
		ReferencedTweets: []V2ReferencedTweet{{Type: "retweeted", ID: "x"}},
	}
	if got := tweetText(tw, nil); got != "fallback" {
		t.Errorf("got %q, want %q", got, "fallback")
	}
}

func TestTweetTextQuoted(t *testing.T) {
	tw := V2Tweet{
		Text:             "my comment",
		ReferencedTweets: []V2ReferencedTweet{{Type: "quoted", ID: "q"}},
	}
	tm := map[string]V2Tweet{"q": {ID: "q", Text: "quoted body"}}
	want := "my comment\n  > quoted body"
	if got := tweetText(tw, tm); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTweetTextRetweetTakesPrecedenceOverQuoted(t *testing.T) {
	tw := V2Tweet{
		Text: "ignored",
		ReferencedTweets: []V2ReferencedTweet{
			{Type: "quoted", ID: "q"},
			{Type: "retweeted", ID: "r"},
		},
	}
	tm := map[string]V2Tweet{
		"q": {ID: "q", Text: "quote"},
		"r": {ID: "r", Text: "retweet"},
	}
	if got := tweetText(tw, tm); got != "RT: retweet" {
		t.Errorf("got %q, want %q", got, "RT: retweet")
	}
}

func TestTweetTextRepliedToFallsThrough(t *testing.T) {
	tw := V2Tweet{
		Text:             "reply body",
		ReferencedTweets: []V2ReferencedTweet{{Type: "replied_to", ID: "p"}},
	}
	if got := tweetText(tw, nil); got != "reply body" {
		t.Errorf("got %q, want %q", got, "reply body")
	}
}
