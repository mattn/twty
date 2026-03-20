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

### All options

    -a PROFILE: switch profile to load configuration file.
    -f ID: specify favorite ID
    -i ID: specify in-reply ID, if not specify text, it will be RT.
    -l LIST: show list's timeline (list ID or user/list-name)
    -m FILE: upload media
    -u USER: show user's timeline
    -s WORD: search timeline
    -S DELAY: tweets after DELAY
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
