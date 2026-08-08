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

### Read a tweet from URL

    $ twty -read https://x.com/USERNAME/status/TWEET_ID
    $ twty -read TWEET_ID

This is handy for AI assistants like Claude: x.com refuses direct fetching,
so let them run `twty -read URL` instead. A profile URL such as
`https://x.com/USERNAME` shows the user's recent tweets, and `t.co`
short URLs are expanded automatically.

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

By default the MCP server is **read-only**: posting tools (`post_tweet`, `like_tweet`, `retweet`) are not exposed, so an AI assistant can never post on your behalf by accident. To opt in to posting, start the server with `-allow-post`:

    $ twty -mcp -allow-post

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
| `read_tweet` | Read a tweet from an X URL or tweet ID |
| `post_tweet` | Post a new tweet (with optional reply) — requires `-allow-post` |
| `like_tweet` | Like a tweet — requires `-allow-post` |
| `retweet` | Retweet a tweet — requires `-allow-post` |

**Note:** You must run `twty` at least once without `-mcp` first to complete OAuth authorization.

### All options

    -a PROFILE: switch profile to load configuration file.
    -f ID: specify favorite ID
    -i ID: specify in-reply ID, if not specify text, it will be RT.
    -l LIST: show list's timeline (list ID or user/list-name)
    -m FILE: upload media
    -u USER: show user's timeline
    -s WORD: search timeline
    -read URL: read tweet(s) from an X URL or tweet ID (status or profile URL)
    -S DELAY: tweets after DELAY
    -mcp: run as MCP server over stdio. Read-only unless -allow-post is given.
    -allow-post: enable posting tools (post_tweet/like_tweet/retweet) in MCP mode.
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

## License

under the MIT License: http://mattn.mit-license.org/2017

## Author

Yasuhiro Matsumoto <mattn.jp@gmail.com>

Have Fun!
