# twty

A command-line twitter client

## Install

1. Install golang environment.

   See: http://golang.org/doc/install.html

2. get twty

        $ go install github.com/mattn/twty@latest

Thanks all!

## Install for Homebrew users (macOS)

    % brew install twty

You would not need to install the golang compiler with this method.

## Usage

    $ twty -h

At the first, you can see the opening web-browser.  And you'll see pin-code is
shown on twitter.com.  Please copy it and type in console like following.

    PIN: XXXXXX

Configuration file is stored in: ~/.config/twty/settings.json
For windows user: %APPDATA%/twty/settings.json

## FAQ

Do you use proxy? then set environment variable `HTTP_PROXY` like below.

    HTTP_PROXY=http://myproxy.example.com:8080

## License

under the MIT License: http://mattn.mit-license.org/2017

## Author

Yasuhiro Matsumoto <mattn.jp@gmail.com>

Have Fun!
