package main

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

func (app *App) fetchHomeTweets(count, sinceID, maxID string) (V2TweetsResponse, error) {
	myID, err := app.getMyID()
	if err != nil {
		return V2TweetsResponse{}, err
	}

	params := v2TweetFields()
	if count != "" {
		params["max_results"] = count
	}
	if sinceID != "" {
		params["since_id"] = sinceID
	}
	if maxID != "" {
		params["until_id"] = maxID
	}

	var res V2TweetsResponse
	err = app.callGet("https://api.twitter.com/2/users/"+myID+"/timelines/reverse_chronological", params, &res)
	return res, err
}

func (app *App) fetchSearchTweets(query, count, since, until, sinceID string) (V2TweetsResponse, error) {
	params := v2TweetFields()
	params["query"] = query
	if count != "" {
		params["max_results"] = count
	}
	if since != "" && isTimeFormat(since) {
		params["start_time"] = since + "T00:00:00Z"
	}
	if until != "" && isTimeFormat(until) {
		params["end_time"] = until + "T00:00:00Z"
	}
	if sinceID != "" {
		params["since_id"] = sinceID
	}

	var res V2TweetsResponse
	err := app.callGet("https://api.twitter.com/2/tweets/search/recent", params, &res)
	return res, err
}

func (app *App) fetchMentions(count string) (V2TweetsResponse, error) {
	myID, err := app.getMyID()
	if err != nil {
		return V2TweetsResponse{}, err
	}

	params := v2TweetFields()
	if count != "" {
		params["max_results"] = count
	}

	var res V2TweetsResponse
	err = app.callGet("https://api.twitter.com/2/users/"+myID+"/mentions", params, &res)
	return res, err
}

func (app *App) fetchUserTweets(username, count, sinceID, maxID string) (V2TweetsResponse, error) {
	var userRes V2UserResponse
	err := app.callGet("https://api.twitter.com/2/users/by/username/"+username, nil, &userRes)
	if err != nil {
		return V2TweetsResponse{}, err
	}

	params := v2TweetFields()
	if count != "" {
		params["max_results"] = count
	}
	if sinceID != "" {
		params["since_id"] = sinceID
	}
	if maxID != "" {
		params["until_id"] = maxID
	}

	var res V2TweetsResponse
	err = app.callGet("https://api.twitter.com/2/users/"+userRes.Data.ID+"/tweets", params, &res)
	return res, err
}

func (app *App) resolveListID(list string) (string, error) {
	if _, err := strconv.ParseInt(list, 10, 64); err == nil {
		return list, nil
	}

	part := strings.SplitN(list, "/", 2)

	var ownerID string
	if len(part) == 1 {
		id, err := app.getMyID()
		if err != nil {
			return "", err
		}
		ownerID = id
	} else {
		var userRes V2UserResponse
		err := app.callGet("https://api.twitter.com/2/users/by/username/"+part[0], nil, &userRes)
		if err != nil {
			return "", err
		}
		ownerID = userRes.Data.ID
	}

	slug := part[len(part)-1]

	var listsRes V2ListsResponse
	err := app.callGet("https://api.twitter.com/2/users/"+ownerID+"/owned_lists", map[string]string{
		"list.fields": "name",
	}, &listsRes)
	if err != nil {
		return "", err
	}

	for _, l := range listsRes.Data {
		if strings.EqualFold(l.Name, slug) {
			return l.ID, nil
		}
	}
	return "", fmt.Errorf("list not found: %s", slug)
}

func (app *App) fetchListTweets(list, count string) (V2TweetsResponse, error) {
	listID, err := app.resolveListID(list)
	if err != nil {
		return V2TweetsResponse{}, err
	}

	params := v2TweetFields()
	if count != "" {
		params["max_results"] = count
	}

	var res V2TweetsResponse
	err = app.callGet("https://api.twitter.com/2/lists/"+listID+"/tweets", params, &res)
	return res, err
}

func (app *App) createTweet(text, inReplyTo string, mediaIDs []string) (string, error) {
	body := map[string]interface{}{
		"text": text,
	}
	if inReplyTo != "" {
		body["reply"] = map[string]string{
			"in_reply_to_tweet_id": inReplyTo,
		}
	}
	if len(mediaIDs) > 0 {
		body["media"] = map[string]interface{}{
			"media_ids": mediaIDs,
		}
	}
	var res V2TweetResponse
	err := app.callPost("https://api.twitter.com/2/tweets", body, &res)
	if err != nil {
		return "", err
	}
	return res.Data.ID, nil
}

func (app *App) likeTweet(tweetID string) error {
	myID, err := app.getMyID()
	if err != nil {
		return err
	}
	body := map[string]string{
		"tweet_id": tweetID,
	}
	return app.callPost("https://api.twitter.com/2/users/"+myID+"/likes", body, nil)
}

func (app *App) retweet(tweetID string) error {
	myID, err := app.getMyID()
	if err != nil {
		return err
	}
	body := map[string]string{
		"tweet_id": tweetID,
	}
	return app.callPost("https://api.twitter.com/2/users/"+myID+"/retweets", body, nil)
}

func formatTweetsText(res V2TweetsResponse) string {
	if len(res.Data) == 0 {
		return "No tweets found."
	}

	userMap := make(map[string]V2User)
	for _, u := range res.Includes.Users {
		userMap[u.ID] = u
	}
	tweetMap := make(map[string]V2Tweet)
	for _, t := range res.Includes.Tweets {
		tweetMap[t.ID] = t
	}

	var sb strings.Builder
	for i := len(res.Data) - 1; i >= 0; i-- {
		tweet := res.Data[i]
		user := userMap[tweet.AuthorID]
		text := tweetText(tweet, tweetMap)
		fmt.Fprintf(&sb, "@%s (%s) [%s]:\n%s\n\n", user.Username, user.Name, tweet.ID, html.UnescapeString(text))
	}
	return strings.TrimSpace(sb.String())
}
