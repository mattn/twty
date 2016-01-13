package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/garyburd/go-oauth/oauth"
)

type Tweet struct {
	Text       string `json:"text"`
	Identifier string `json:"id_str"`
	Source     string `json:"source"`
	CreatedAt  string `json:"created_at"`
	User       struct {
		Name            string `json:"name"`
		ScreenName      string `json:"screen_name"`
		FollowersCount  int    `json:"followers_count"`
		ProfileImageURL string `json:"profile_image_url"`
	} `json:"user"`
	Place *struct {
		Id       string `json:"id"`
		FullName string `json:"full_name"`
	} `json:"place"`
	Entities struct {
		HashTags []struct {
			Indices [2]int `json:"indices"`
			Text    string `json:"text"`
		}
		UserMentions []struct {
			Indices    [2]int `json:"indices"`
			ScreenName string `json:"screen_name"`
		} `json:"user_mentions"`
		Urls []struct {
			Indices [2]int `json:"indices"`
			Url     string `json:"url"`
		} `json:"urls"`
	} `json:"entities"`
}

type RSS struct {
	Channel struct {
		Title       string
		Description string
		Link        string
		Item        []struct {
			Title       string
			Description string
			PubDate     string
			Link        []string
			Guid        string
			Author      string
		}
	}
}

var oauthClient = oauth.Client{
	TemporaryCredentialRequestURI: "https://api.twitter.com/oauth/request_token",
	ResourceOwnerAuthorizationURI: "https://api.twitter.com/oauth/authenticate",
	TokenRequestURI:               "https://api.twitter.com/oauth/access_token",
}

func clientAuth(requestToken *oauth.Credentials) (*oauth.Credentials, error) {
	var err error
	browser := "xdg-open"
	url_ := oauthClient.AuthorizationURL(requestToken, nil)

	args := []string{url_}
	if runtime.GOOS == "windows" {
		browser = "rundll32.exe"
		args = []string{"url.dll,FileProtocolHandler", url_}
	} else if runtime.GOOS == "darwin" {
		browser = "open"
		args = []string{url_}
	} else if runtime.GOOS == "plan9" {
		browser = "plumb"
	}
	color.Set(color.FgHiRed)
	fmt.Println("Open this URL and enter PIN.")
	color.Set(color.Reset)
	fmt.Println(url_)
	browser, err = exec.LookPath(browser)
	if err == nil {
		cmd := exec.Command(browser, args...)
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			log.Fatal("failed to start command:", err)
		}
	}

	fmt.Print("PIN: ")
	stdin := bufio.NewScanner(os.Stdin)
	if !stdin.Scan() {
		log.Fatal("canceled")
	}
	accessToken, _, err := oauthClient.RequestToken(http.DefaultClient, requestToken, stdin.Text())
	if err != nil {
		log.Fatal("failed to request token:", err)
	}
	return accessToken, nil
}

func getAccessToken(config map[string]string) (*oauth.Credentials, bool, error) {
	oauthClient.Credentials.Token = config["ClientToken"]
	oauthClient.Credentials.Secret = config["ClientSecret"]

	authorized := false
	var token *oauth.Credentials
	accessToken, foundToken := config["AccessToken"]
	accessSecert, foundSecret := config["AccessSecret"]
	if foundToken && foundSecret {
		token = &oauth.Credentials{accessToken, accessSecert}
	} else {
		requestToken, err := oauthClient.RequestTemporaryCredentials(http.DefaultClient, "", nil)
		if err != nil {
			log.Print("failed to request temporary credentials:", err)
			return nil, false, err
		}
		token, err = clientAuth(requestToken)
		if err != nil {
			log.Print("failed to request temporary credentials:", err)
			return nil, false, err
		}

		config["AccessToken"] = token.Token
		config["AccessSecret"] = token.Secret
		authorized = true
	}
	return token, authorized, nil
}

func getTweets(token *oauth.Credentials, url_ string, opt map[string]string) ([]Tweet, error) {
	param := make(url.Values)
	for k, v := range opt {
		param.Set(k, v)
	}
	oauthClient.SignParam(token, "GET", url_, param)
	url_ = url_ + "?" + param.Encode()
	res, err := http.Get(url_)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}
	var tweets []Tweet
	err = json.NewDecoder(res.Body).Decode(&tweets)
	return tweets, err
}

func getStatuses(token *oauth.Credentials, url_ string, opt map[string]string) ([]Tweet, error) {
	param := make(url.Values)
	for k, v := range opt {
		param.Set(k, v)
	}
	oauthClient.SignParam(token, "GET", url_, param)
	url_ = url_ + "?" + param.Encode()
	res, err := http.Get(url_)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}
	var statuses struct {
		Statuses []Tweet
	}
	err = json.NewDecoder(res.Body).Decode(&statuses)
	return statuses.Statuses, err
}

var replacer = strings.NewReplacer(
	"\r", "",
	"\n", " ",
	"\t", " ",
)

func showTweets(tweets []Tweet, verbose bool) {
	if verbose {
		for i := len(tweets) - 1; i >= 0; i-- {
			name := tweets[i].User.Name
			user := tweets[i].User.ScreenName
			text := tweets[i].Text
			text = replacer.Replace(text)
			color.Set(color.FgHiGreen)
			fmt.Println(user + ": " + name)
			color.Set(color.FgHiWhite)
			fmt.Println("  " + text)
			fmt.Println("  " + tweets[i].Identifier)
			fmt.Println("  " + tweets[i].CreatedAt)
			fmt.Println()
			color.Set(color.Reset)
		}
	} else {
		for i := len(tweets) - 1; i >= 0; i-- {
			user := tweets[i].User.ScreenName
			text := tweets[i].Text
			color.Set(color.FgHiGreen)
			fmt.Print(user)
			color.Set(color.Reset)
			fmt.Print(": ")
			color.Set(color.FgHiWhite)
			fmt.Println(text)
			color.Set(color.Reset)
		}
	}
}

func postTweet(token *oauth.Credentials, url_ string, opt map[string]string) error {
	param := make(url.Values)
	for k, v := range opt {
		param.Set(k, v)
	}
	oauthClient.SignParam(token, "POST", url_, param)
	res, err := http.PostForm(url_, url.Values(param))
	if err != nil {
		log.Println("failed to post tweet:", err)
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Println("failed to get timeline:", err)
		return err
	}
	var tweet Tweet
	err = json.NewDecoder(res.Body).Decode(&tweet)
	if err != nil {
		log.Println("failed to parse new tweet:", err)
		return err
	}
	fmt.Println("tweeted:", tweet.Identifier)
	return nil
}

func getConfig() (string, map[string]string) {
	home := os.Getenv("HOME")
	dir := filepath.Join(home, ".config")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
		dir = os.Getenv("APPDATA")
		if dir == "" {
			dir = filepath.Join(home, "Application Data")
		}
	} else if runtime.GOOS == "plan9" {
		home = os.Getenv("home")
		dir = filepath.Join(home, ".config")
	}
	_, err := os.Stat(dir)
	if err != nil {
		if os.Mkdir(dir, 0700) != nil {
			log.Fatal("failed to create directory:", err)
		}
	}
	dir = filepath.Join(dir, "twty")
	_, err = os.Stat(dir)
	if err != nil {
		if os.Mkdir(dir, 0700) != nil {
			log.Fatal("failed to create directory:", err)
		}
	}
	var file string
	if *account == "" {
		file = filepath.Join(dir, "settings.json")
	} else {
		file = filepath.Join(dir, "settings-"+*account+".json")
	}
	config := map[string]string{}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		config["ClientToken"] = "MbartJkKCrSegn45xK9XLw"
		config["ClientSecret"] = "1nI3dHFtK9UY1kL6UEYWk6r2lFEcNHWhk7MtXe7eo"
	} else {
		err = json.Unmarshal(b, &config)
		if err != nil {
			log.Fatalf("could not unmarhal %v: %v", file, err)
		}
	}
	return file, config
}

var (
	account  = flag.String("a", "", "account")
	reply    = flag.Bool("r", false, "show replies")
	list     = flag.String("l", "", "show tweets")
	user     = flag.String("u", "", "show user timeline")
	favorite = flag.String("f", "", "specify favorite ID")
	search   = flag.String("s", "", "search word")
	stream   = flag.Bool("S", false, "stream timeline")
	inreply  = flag.String("i", "", "specify in-reply ID, if not specify text, it will be RT.")
	verbose  = flag.Bool("v", false, "detail display")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage of twty:
  -a ACCOUNT: switch account to load configuration file. Note: experimental
  -f ID: specify favorite ID
  -i ID: specify in-reply ID, if not specify text, it will be RT.
  -l USER/LIST: show list's timeline (ex: mattn_jp/subtech)
  -u USER: show user's timeline
  -s WORD: search timeline
  -S: stream timeline
  -r: show replies
  -v: detail display
`)
	}
	flag.Parse()

	http.DefaultTransport.(*http.Transport).DisableCompression = true

	file, config := getConfig()
	token, authorized, err := getAccessToken(config)
	if err != nil {
		log.Fatal("faild to get access token:", err)
	}
	if authorized {
		b, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			log.Fatal("failed to store file:", err)
		}
		err = ioutil.WriteFile(file, b, 0700)
		if err != nil {
			log.Fatal("failed to store file:", err)
		}
	}

	if len(*search) > 0 {
		tweets, err := getStatuses(token, "https://api.twitter.com/1.1/search/tweets.json", map[string]string{"q": *search})
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		showTweets(tweets, *verbose)
	} else if *reply {
		tweets, err := getTweets(token, "https://api.twitter.com/1.1/statuses/mentions_timeline.json", map[string]string{})
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		showTweets(tweets, *verbose)
	} else if len(*list) > 0 {
		part := strings.SplitN(*list, "/", 2)
		if len(part) == 2 {
			tweets, err := getTweets(token, "https://api.twitter.com/1.1/lists/statuses.json", map[string]string{"owner_screen_name": part[0], "slug": part[1]})
			if err != nil {
				log.Fatal("failed to get tweets:", err)
			}
			showTweets(tweets, *verbose)
		}
	} else if len(*user) > 0 {
		tweets, err := getTweets(token, "https://api.twitter.com/1.1/statuses/user_timeline.json", map[string]string{"screen_name": *user})
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		showTweets(tweets, *verbose)
	} else if len(*favorite) > 0 {
		postTweet(token, "https://api.twitter.com/1.1/favorites/create.json", map[string]string{"id": *favorite})
	} else if *stream {
		url_ := "https://userstream.twitter.com/1.1/user.json"
		param := make(url.Values)
		oauthClient.SignParam(token, "GET", url_, param)
		url_ = url_ + "?" + param.Encode()
		res, err := http.Get(url_)
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			log.Fatal("failed to get tweets:", err)
		}
		var buf *bufio.Reader
		if res.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(res.Body)
			if err != nil {
				log.Fatal("failed to make gzip decoder:", err)
			}
			buf = bufio.NewReader(gr)
		} else {
			buf = bufio.NewReader(res.Body)
		}
		var last []byte
		for {
			b, _, err := buf.ReadLine()
			last = append(last, b...)
			var tweets [1]Tweet
			err = json.Unmarshal(last, &tweets[0])
			if err != nil {
				continue
			}
			last = []byte{}
			if tweets[0].Identifier != "" {
				showTweets(tweets[:], *verbose)
			}
		}
	} else if flag.NArg() == 0 {
		if len(*inreply) > 0 {
			postTweet(token, "https://api.twitter.com/1.1/statuses/retweet/"+*inreply+".json", map[string]string{})
		} else {
			tweets, err := getTweets(token, "https://api.twitter.com/1.1/statuses/home_timeline.json", map[string]string{})
			if err != nil {
				log.Fatal("failed to get tweets:", err)
			}
			showTweets(tweets, *verbose)
		}
	} else {
		postTweet(token, "https://api.twitter.com/1.1/statuses/update.json", map[string]string{"status": strings.Join(flag.Args(), " "), "in_reply_to_status_id": *inreply})
	}
}
