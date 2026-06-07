# twty

A command-line client for X (formerly Twitter)

## Install

1. Install golang environment.

   See: http://golang.org/doc/install.html

2. get twty

        $ go install github.com/mattn/twty@latest

Thanks all!

## Install for Homebrew users (macOS)

    % brew install twty

You would not need to install the golang compiler with this method.

## Setup

twty uses X API v2 with OAuth 2.0 authentication.

1. Create an app at the [X Developer Portal](https://developer.x.com/en/portal/dashboard)
2. Set the callback URL to `http://localhost:8989/callback` in your app settings
3. Run twty — a browser window will open for authorization

Configuration file is stored in: `~/.config/twty/settings.json`
For windows user: `%APPDATA%/twty/settings.json`

## Usage

    $ twty -h

### Post a tweet

    $ twty Hello World

### Show home timeline

    $ twty

### Show user timeline

    $ twty -u USERNAME

### Search tweets

    $ twty -s KEYWORD

### Show replies/mentions

    $ twty -r

### Like a tweet

    $ twty -f TWEET_ID

### Retweet

    $ twty -i TWEET_ID

### Reply to a tweet

    $ twty -i TWEET_ID Your reply here

### Post with media

    $ twty -m image.png Hello with image

### Show list timeline

    $ twty -l LIST_ID
    $ twty -l USERNAME/LIST_NAME

### Polling mode

    $ twty -S 60s

### MCP server mode

twty can run as an [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) server, allowing AI assistants like Claude to interact with X directly.

    $ twty -mcp

To use with Claude Code, add the following to your MCP settings:

```json
{
  "mcpServers": {
    "twty": {
      "command": "twty",
      "args": ["-mcp"]
    }
  }
}
```

Available tools:

| Tool | Description |
|------|-------------|
| `get_timeline` | Get your home timeline |
| `search_tweets` | Search recent tweets |
| `get_mentions` | Get your mentions and replies |
| `get_user_tweets` | Get tweets from a specific user |
| `get_list_tweets` | Get tweets from a list |
| `post_tweet` | Post a new tweet (with optional reply) |
| `like_tweet` | Like a tweet |
| `retweet` | Retweet a tweet |

**Note:** You must run `twty` at least once without `-mcp` first to complete OAuth authorization.

### All options

    -a PROFILE: switch profile to load configuration file.
    -f ID: specify favorite ID
    -i ID: specify in-reply ID, if not specify text, it will be RT.
    -l LIST: show list's timeline (list ID or user/list-name)
    -m FILE: upload media
    -u USER: show user's timeline
    -s WORD: search timeline
    -S DELAY: tweets after DELAY
    -mcp: run as MCP server
    -json: as JSON
    -r: show replies
    -v: detail display
    -ff FILENAME: post utf-8 string from a file("-" means STDIN)
    -count NUMBER: show NUMBER tweets at timeline.
    -since DATE: show tweets created after the DATE (ex. 2017-05-01)
    -until DATE: show tweets created before the DATE (ex. 2017-05-31)
    -since_id NUMBER: show tweets that have ids greater than NUMBER.
    -max_id NUMBER: show tweets that have ids lower than NUMBER.

## FAQ

Do you use proxy? then set environment variable `HTTP_PROXY` like below.

    HTTP_PROXY=http://myproxy.example.com:8080

## Pairing with GetXAPI for Cheaper Read Operations (Optional)

For users who need a cheaper or higher-rate-limit option for read-only Twitter (X) operations such as tweet search, profile lookup, and follower lists, this project can be paired with [GetXAPI](https://getxapi.com), a budget Twitter / X data API priced at $0.05 per 1K tweets versus the official X API basic tier at $200 / month.

Two integration patterns:

1. **Run side-by-side in your AI client.** Keep this project for its primary workflow and add the [official GetXAPI MCP server](https://github.com/getxapi/getxapi-mcp) for read-heavy tasks. Each tool name routes to the backend best suited for that operation.

2. **Add a backend toggle.** For a code-level reference of an optional alternative backend behind a single env variable, see the [PR pattern merged into a sibling project](https://github.com/GenAIwithMS/twitter-mcp/pull/3).

GetXAPI quick start:

- Signup with $0.50 free credit (no card required): https://getxapi.com/signup
- Official GetXAPI MCP server: https://github.com/getxapi/getxapi-mcp
- npm: `@getxapi/mcp`
- Pay-per-call pricing: $0.001 / call, $0.05 / 1K tweets

This pairing is fully optional. No behavior change for existing users.

## License

under the MIT License: http://mattn.mit-license.org/2017

## Author

Yasuhiro Matsumoto <mattn.jp@gmail.com>

Have Fun!
