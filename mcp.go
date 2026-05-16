package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
)

type jsonrpcRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *jsonrpcError    `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpToolResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

var mcpTools = []mcpTool{
	{
		Name:        "get_timeline",
		Description: "Get your X (Twitter) home timeline",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"count":{"type":"string","description":"Number of tweets to fetch (max 100)"}}}`),
	},
	{
		Name:        "search_tweets",
		Description: "Search recent tweets on X (Twitter)",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query"},"count":{"type":"string","description":"Number of tweets to fetch (max 100)"}},"required":["query"]}`),
	},
	{
		Name:        "get_mentions",
		Description: "Get your mentions and replies on X (Twitter)",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"count":{"type":"string","description":"Number of tweets to fetch (max 100)"}}}`),
	},
	{
		Name:        "get_user_tweets",
		Description: "Get tweets from a specific user on X (Twitter)",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"username":{"type":"string","description":"Twitter username (without @)"},"count":{"type":"string","description":"Number of tweets to fetch (max 100)"}},"required":["username"]}`),
	},
	{
		Name:        "get_list_tweets",
		Description: "Get tweets from a list on X (Twitter)",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"list":{"type":"string","description":"List ID or owner/list-name"},"count":{"type":"string","description":"Number of tweets to fetch (max 100)"}},"required":["list"]}`),
	},
	{
		Name:        "post_tweet",
		Description: "Post a new tweet on X (Twitter)",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","description":"Tweet text"},"reply_to":{"type":"string","description":"Tweet ID to reply to"}},"required":["text"]}`),
	},
	{
		Name:        "like_tweet",
		Description: "Like (favorite) a tweet on X (Twitter)",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"tweet_id":{"type":"string","description":"Tweet ID to like"}},"required":["tweet_id"]}`),
	},
	{
		Name:        "retweet",
		Description: "Retweet a tweet on X (Twitter)",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"tweet_id":{"type":"string","description":"Tweet ID to retweet"}},"required":["tweet_id"]}`),
	},
}

func (app *App) serveMCP() {
	enc := json.NewEncoder(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	const maxMCPMessageSize = 16 * 1024 * 1024
	scanner.Buffer(make([]byte, 64*1024), maxMCPMessageSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("invalid JSON-RPC request: %v", err)
			continue
		}

		if req.ID == nil {
			continue
		}

		resp := jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		switch req.Method {
		case "initialize":
			resp.Result = mustMarshal(map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    name,
					"version": version,
				},
			})
		case "tools/list":
			resp.Result = mustMarshal(map[string]interface{}{
				"tools": mcpTools,
			})
		case "tools/call":
			result, rpcErr := app.handleToolCall(req.Params)
			if rpcErr != nil {
				resp.Error = rpcErr
			} else {
				resp.Result = mustMarshal(result)
			}
		default:
			resp.Error = &jsonrpcError{Code: -32601, Message: "method not found: " + req.Method}
		}

		if err := enc.Encode(resp); err != nil {
			log.Printf("cannot encode response: %v", err)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("scanner error: %v", err)
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("marshal error: %v", err)
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(b)
}

func (app *App) handleToolCall(params json.RawMessage) (*mcpToolResult, *jsonrpcError) {
	var req struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, &jsonrpcError{Code: -32602, Message: "invalid params: " + err.Error()}
	}

	switch req.Name {
	case "get_timeline":
		return app.mcpGetTimeline(req.Arguments)
	case "search_tweets":
		return app.mcpSearchTweets(req.Arguments)
	case "get_mentions":
		return app.mcpGetMentions(req.Arguments)
	case "get_user_tweets":
		return app.mcpGetUserTweets(req.Arguments)
	case "get_list_tweets":
		return app.mcpGetListTweets(req.Arguments)
	case "post_tweet":
		return app.mcpPostTweet(req.Arguments)
	case "like_tweet":
		return app.mcpLikeTweet(req.Arguments)
	case "retweet":
		return app.mcpRetweet(req.Arguments)
	default:
		return nil, &jsonrpcError{Code: -32602, Message: "unknown tool: " + req.Name}
	}
}

func textResult(text string) *mcpToolResult {
	return &mcpToolResult{
		Content: []mcpContent{{Type: "text", Text: text}},
	}
}

func errorResult(err error) *mcpToolResult {
	return &mcpToolResult{
		Content: []mcpContent{{Type: "text", Text: err.Error()}},
		IsError: true,
	}
}

func (app *App) mcpGetTimeline(args json.RawMessage) (*mcpToolResult, *jsonrpcError) {
	var p struct {
		Count string `json:"count"`
	}
	json.Unmarshal(args, &p)

	res, err := app.fetchHomeTweets(p.Count, "", "")
	if err != nil {
		return errorResult(err), nil
	}
	return textResult(formatTweetsText(res)), nil
}

func (app *App) mcpSearchTweets(args json.RawMessage) (*mcpToolResult, *jsonrpcError) {
	var p struct {
		Query string `json:"query"`
		Count string `json:"count"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, &jsonrpcError{Code: -32602, Message: "invalid arguments"}
	}
	if p.Query == "" {
		return nil, &jsonrpcError{Code: -32602, Message: "query is required"}
	}

	res, err := app.fetchSearchTweets(p.Query, p.Count, "", "", "")
	if err != nil {
		return errorResult(err), nil
	}
	return textResult(formatTweetsText(res)), nil
}

func (app *App) mcpGetMentions(args json.RawMessage) (*mcpToolResult, *jsonrpcError) {
	var p struct {
		Count string `json:"count"`
	}
	json.Unmarshal(args, &p)

	res, err := app.fetchMentions(p.Count)
	if err != nil {
		return errorResult(err), nil
	}
	return textResult(formatTweetsText(res)), nil
}

func (app *App) mcpGetUserTweets(args json.RawMessage) (*mcpToolResult, *jsonrpcError) {
	var p struct {
		Username string `json:"username"`
		Count    string `json:"count"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, &jsonrpcError{Code: -32602, Message: "invalid arguments"}
	}
	if p.Username == "" {
		return nil, &jsonrpcError{Code: -32602, Message: "username is required"}
	}

	res, err := app.fetchUserTweets(p.Username, p.Count, "", "")
	if err != nil {
		return errorResult(err), nil
	}
	return textResult(formatTweetsText(res)), nil
}

func (app *App) mcpGetListTweets(args json.RawMessage) (*mcpToolResult, *jsonrpcError) {
	var p struct {
		List  string `json:"list"`
		Count string `json:"count"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, &jsonrpcError{Code: -32602, Message: "invalid arguments"}
	}
	if p.List == "" {
		return nil, &jsonrpcError{Code: -32602, Message: "list is required"}
	}

	res, err := app.fetchListTweets(p.List, p.Count)
	if err != nil {
		return errorResult(err), nil
	}
	return textResult(formatTweetsText(res)), nil
}

func (app *App) mcpPostTweet(args json.RawMessage) (*mcpToolResult, *jsonrpcError) {
	var p struct {
		Text    string `json:"text"`
		ReplyTo string `json:"reply_to"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, &jsonrpcError{Code: -32602, Message: "invalid arguments"}
	}
	if p.Text == "" {
		return nil, &jsonrpcError{Code: -32602, Message: "text is required"}
	}

	id, err := app.createTweet(p.Text, p.ReplyTo, nil)
	if err != nil {
		return errorResult(err), nil
	}
	return textResult("tweeted: " + id), nil
}

func (app *App) mcpLikeTweet(args json.RawMessage) (*mcpToolResult, *jsonrpcError) {
	var p struct {
		TweetID string `json:"tweet_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, &jsonrpcError{Code: -32602, Message: "invalid arguments"}
	}
	if p.TweetID == "" {
		return nil, &jsonrpcError{Code: -32602, Message: "tweet_id is required"}
	}

	if err := app.likeTweet(p.TweetID); err != nil {
		return errorResult(err), nil
	}
	return textResult("liked: " + p.TweetID), nil
}

func (app *App) mcpRetweet(args json.RawMessage) (*mcpToolResult, *jsonrpcError) {
	var p struct {
		TweetID string `json:"tweet_id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, &jsonrpcError{Code: -32602, Message: "invalid arguments"}
	}
	if p.TweetID == "" {
		return nil, &jsonrpcError{Code: -32602, Message: "tweet_id is required"}
	}

	if err := app.retweet(p.TweetID); err != nil {
		return errorResult(err), nil
	}
	return textResult("retweeted: " + p.TweetID), nil
}
