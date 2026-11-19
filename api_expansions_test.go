package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type fakeRoundTripper struct {
	t        *testing.T
	requests []*url.URL
	handle   func(u *url.URL) string
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.requests = append(f.requests, req.URL)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(f.handle(req.URL))),
		Request:    req,
	}, nil
}

func newFakeAPI(t *testing.T, handle func(u *url.URL) string) (*App, *fakeRoundTripper) {
	t.Helper()
	rt := &fakeRoundTripper{t: t, handle: handle}
	orig := http.DefaultClient.Transport
	http.DefaultClient.Transport = rt
	t.Cleanup(func() { http.DefaultClient.Transport = orig })
	app := &App{}
	app.config.Token.AccessToken = "token"
	app.config.Token.ExpiresAt = time.Now().Add(time.Hour)
	return app, rt
}

func TestReferencedTweetIDsToFetch(t *testing.T) {
	res := V2TweetsResponse{
		Data: []V2Tweet{
			{ID: "1", ReferencedTweets: []V2ReferencedTweet{{Type: "retweeted", ID: "10"}}},
			{ID: "2", ReferencedTweets: []V2ReferencedTweet{{Type: "replied_to", ID: "20"}}},
			{ID: "3", ReferencedTweets: []V2ReferencedTweet{{Type: "quoted", ID: "30"}, {Type: "replied_to", ID: "31"}}},
			{ID: "4", ReferencedTweets: []V2ReferencedTweet{{Type: "retweeted", ID: "10"}}},
			{ID: "5", ReferencedTweets: []V2ReferencedTweet{{Type: "quoted", ID: "50"}}},
		},
		Includes: V2Includes{Tweets: []V2Tweet{{ID: "50"}}},
	}
	got := strings.Join(referencedTweetIDsToFetch(res), ",")
	if got != "10,30" {
		t.Errorf("got %q, want %q", got, "10,30")
	}
}

func TestFetchHomeTweetsLooksUpOnlyDisplayedReferences(t *testing.T) {
	app, rt := newFakeAPI(t, func(u *url.URL) string {
		switch u.Path {
		case "/2/users/me":
			return `{"data":{"id":"me"}}`
		case "/2/users/me/timelines/reverse_chronological":
			return `{"data":[
				{"id":"1","author_id":"a","text":"RT @b: trunc…","referenced_tweets":[{"type":"retweeted","id":"10"}]},
				{"id":"2","author_id":"a","text":"reply","referenced_tweets":[{"type":"replied_to","id":"20"}]}
			],"includes":{"users":[{"id":"a","username":"alice","name":"Alice"}]}}`
		case "/2/tweets":
			return `{"data":[{"id":"10","text":"full original"}]}`
		}
		t.Fatalf("unexpected request: %s", u)
		return ""
	})

	res, err := app.fetchHomeTweets("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rt.requests) != 3 {
		t.Fatalf("got %d requests, want 3", len(rt.requests))
	}
	if got := rt.requests[1].Query().Get("expansions"); got != "author_id" {
		t.Errorf("timeline expansions = %q, want %q", got, "author_id")
	}
	lookup := rt.requests[2].Query()
	if got := lookup.Get("ids"); got != "10" {
		t.Errorf("lookup ids = %q, want %q", got, "10")
	}
	if lookup.Has("expansions") {
		t.Errorf("lookup must not request expansions, got %q", lookup.Get("expansions"))
	}
	if got := formatTweetsText(res); !strings.Contains(got, "RT: full original") {
		t.Errorf("formatted text missing retweet body: %q", got)
	}
}

func TestFetchHomeTweetsSkipsLookupWithoutReferences(t *testing.T) {
	app, rt := newFakeAPI(t, func(u *url.URL) string {
		switch u.Path {
		case "/2/users/me":
			return `{"data":{"id":"me"}}`
		case "/2/users/me/timelines/reverse_chronological":
			return `{"data":[{"id":"1","author_id":"a","text":"hi"}]}`
		}
		t.Fatalf("unexpected request: %s", u)
		return ""
	})
	if _, err := app.fetchHomeTweets("", "", ""); err != nil {
		t.Fatal(err)
	}
	if len(rt.requests) != 2 {
		t.Errorf("got %d requests, want 2", len(rt.requests))
	}
}

func TestFetchUserTweetsDoesNotExpandAuthor(t *testing.T) {
	app, rt := newFakeAPI(t, func(u *url.URL) string {
		switch u.Path {
		case "/2/users/by/username/alice":
			return `{"data":{"id":"a","username":"alice","name":"Alice"}}`
		case "/2/users/a/tweets":
			return `{"data":[{"id":"1","author_id":"a","text":"hello"}]}`
		}
		t.Fatalf("unexpected request: %s", u)
		return ""
	})

	res, err := app.fetchUserTweets("alice", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if q := rt.requests[1].Query(); q.Has("expansions") {
		t.Errorf("user timeline must not request expansions, got %q", q.Get("expansions"))
	}
	if got := formatTweetsText(res); !strings.HasPrefix(got, "@alice (Alice) [1]:") {
		t.Errorf("author not filled from user lookup: %q", got)
	}
}
