package main

import (
	"encoding/json"
	"testing"
)

func TestTextResult(t *testing.T) {
	r := textResult("hello")
	if r.IsError {
		t.Errorf("textResult should not be an error")
	}
	if len(r.Content) != 1 || r.Content[0].Type != "text" || r.Content[0].Text != "hello" {
		t.Errorf("unexpected content: %#v", r.Content)
	}
}

func TestErrorResult(t *testing.T) {
	r := errorResult(errString("boom"))
	if !r.IsError {
		t.Errorf("errorResult should set IsError")
	}
	if r.Content[0].Text != "boom" {
		t.Errorf("got %q, want %q", r.Content[0].Text, "boom")
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestMustMarshalRoundtrip(t *testing.T) {
	raw := mustMarshal(map[string]string{"k": "v"})
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["k"] != "v" {
		t.Errorf("got %v, want k=v", out)
	}
}

func TestHandleToolCallUnknown(t *testing.T) {
	app := &App{}
	_, rpcErr := app.handleToolCall(json.RawMessage(`{"name":"nope","arguments":{}}`))
	if rpcErr == nil {
		t.Fatalf("expected error for unknown tool")
	}
	if rpcErr.Code != -32602 {
		t.Errorf("got code %d, want -32602", rpcErr.Code)
	}
}

func TestHandleToolCallBadJSON(t *testing.T) {
	app := &App{}
	_, rpcErr := app.handleToolCall(json.RawMessage(`not json`))
	if rpcErr == nil {
		t.Fatalf("expected error for invalid params")
	}
	if rpcErr.Code != -32602 {
		t.Errorf("got code %d, want -32602", rpcErr.Code)
	}
}

func TestMcpPostTweetMissingText(t *testing.T) {
	app := &App{}
	_, rpcErr := app.mcpPostTweet(json.RawMessage(`{}`))
	if rpcErr == nil || rpcErr.Code != -32602 {
		t.Fatalf("expected -32602 error, got %+v", rpcErr)
	}
}

func TestMcpSearchTweetsMissingQuery(t *testing.T) {
	app := &App{}
	_, rpcErr := app.mcpSearchTweets(json.RawMessage(`{}`))
	if rpcErr == nil || rpcErr.Code != -32602 {
		t.Fatalf("expected -32602 error, got %+v", rpcErr)
	}
}

func TestMcpLikeTweetMissingID(t *testing.T) {
	app := &App{}
	_, rpcErr := app.mcpLikeTweet(json.RawMessage(`{}`))
	if rpcErr == nil || rpcErr.Code != -32602 {
		t.Fatalf("expected -32602 error, got %+v", rpcErr)
	}
}

func TestMcpRetweetMissingID(t *testing.T) {
	app := &App{}
	_, rpcErr := app.mcpRetweet(json.RawMessage(`{}`))
	if rpcErr == nil || rpcErr.Code != -32602 {
		t.Fatalf("expected -32602 error, got %+v", rpcErr)
	}
}

func TestMcpGetUserTweetsMissingUsername(t *testing.T) {
	app := &App{}
	_, rpcErr := app.mcpGetUserTweets(json.RawMessage(`{}`))
	if rpcErr == nil || rpcErr.Code != -32602 {
		t.Fatalf("expected -32602 error, got %+v", rpcErr)
	}
}

func TestMcpGetListTweetsMissingList(t *testing.T) {
	app := &App{}
	_, rpcErr := app.mcpGetListTweets(json.RawMessage(`{}`))
	if rpcErr == nil || rpcErr.Code != -32602 {
		t.Fatalf("expected -32602 error, got %+v", rpcErr)
	}
}

func TestMcpToolsSchemaParses(t *testing.T) {
	for _, tool := range mcpTools {
		var schema map[string]any
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Errorf("tool %s: schema not valid JSON: %v", tool.Name, err)
		}
		if schema["type"] != "object" {
			t.Errorf("tool %s: schema type = %v, want object", tool.Name, schema["type"])
		}
	}
}
