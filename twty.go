package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

type Account struct {
	TimeZone struct {
		Name       string `json:"name"`
		UtcOffset  int    `json:"utc_offset"`
		TzinfoName string `json:"tzinfo_name"`
	} `json:"time_zone"`
	Protected                bool   `json:"protected"`
	ScreenName               string `json:"screen_name"`
	AlwaysUseHTTPS           bool   `json:"always_use_https"`
	UseCookiePersonalization bool   `json:"use_cookie_personalization"`
	SleepTime                struct {
		Enabled   bool        `json:"enabled"`
		EndTime   interface{} `json:"end_time"`
		StartTime interface{} `json:"start_time"`
	} `json:"sleep_time"`
	GeoEnabled                bool   `json:"geo_enabled"`
	Language                  string `json:"language"`
	DiscoverableByEmail       bool   `json:"discoverable_by_email"`
	DiscoverableByMobilePhone bool   `json:"discoverable_by_mobile_phone"`
	DisplaySensitiveMedia     bool   `json:"display_sensitive_media"`
	AllowContributorRequest   string `json:"allow_contributor_request"`
	AllowDmsFrom              string `json:"allow_dms_from"`
	AllowDmGroupsFrom         string `json:"allow_dm_groups_from"`
	SmartMute                 bool   `json:"smart_mute"`
	TrendLocation             []struct {
		Name        string `json:"name"`
		CountryCode string `json:"countryCode"`
		URL         string `json:"url"`
		Woeid       int    `json:"woeid"`
		PlaceType   struct {
			Name string `json:"name"`
			Code int    `json:"code"`
		} `json:"placeType"`
		Parentid int    `json:"parentid"`
		Country  string `json:"country"`
	} `json:"trend_location"`
}

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

type SearchMetadata struct {
	CompletedIn float64 `json:"completed_in"`
	MaxID       int64   `json:"max_id"`
	MaxIDStr    string  `json:"max_id_str"`
	NextResults string  `json:"next_results"`
	Query       string  `json:"query"`
	RefreshURL  string  `json:"refresh_url"`
	Count       int     `json:"count"`
	SinceID     int     `json:"since_id"`
	SinceIDStr  string  `json:"since_id_str"`
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
			return nil, fmt.Errorf("failed to start command:", err)
		}
	}

	fmt.Print("PIN: ")
	stdin := bufio.NewScanner(os.Stdin)
	if !stdin.Scan() {
		return nil, fmt.Errorf("canceled")
	}
	accessToken, _, err := oauthClient.RequestToken(http.DefaultClient, requestToken, stdin.Text())
	if err != nil {
		return nil, fmt.Errorf("failed to request token:", err)
	}
	return accessToken, nil
}

func getAccessToken(config map[string]string) (*oauth.Credentials, bool, error) {
	oauthClient.Credentials.Token = config["ClientToken"]
	oauthClient.Credentials.Secret = config["ClientSecret"]

	authorized := false
	var token *oauth.Credentials
	accessToken, foundToken := config["AccessToken"]
	accessSecret, foundSecret := config["AccessSecret"]
	if foundToken && foundSecret {
		token = &oauth.Credentials{accessToken, accessSecret}
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

func rawCall(token *oauth.Credentials, method string, url_ string, opt map[string]string, res interface{}) error {
	param := make(url.Values)
	for k, v := range opt {
		param.Set(k, v)
	}
	oauthClient.SignParam(token, method, url_, param)
	var resp *http.Response
	var err error
	if method == "GET" {
		url_ = url_ + "?" + param.Encode()
		resp, err = http.Get(url_)
	} else {
		resp, err = http.PostForm(url_, url.Values(param))
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if res == nil {
		return nil
	}
	if *debug {
		return json.NewDecoder(io.TeeReader(resp.Body, os.Stdout)).Decode(&res)
	}
	return json.NewDecoder(resp.Body).Decode(&res)
}

var replacer = strings.NewReplacer(
	"\r", "",
	"\n", " ",
	"\t", " ",
)

func showTweets(tweets []Tweet, verbose bool) {
	if *asjson {
		json.NewEncoder(os.Stdout).Encode(tweets)
	} else if verbose {
		for i := len(tweets) - 1; i >= 0; i-- {
			name := tweets[i].User.Name
			user := tweets[i].User.ScreenName
			text := tweets[i].Text
			text = replacer.Replace(text)
			color.Set(color.FgHiRed)
			fmt.Println(user + ": " + name)
			color.Set(color.Reset)
			fmt.Println("  " + text)
			fmt.Println("  " + tweets[i].Identifier)
			fmt.Println("  " + tweets[i].CreatedAt)
			fmt.Println()
		}
	} else {
		for i := len(tweets) - 1; i >= 0; i-- {
			user := tweets[i].User.ScreenName
			text := tweets[i].Text
			color.Set(color.FgHiRed)
			fmt.Print(user)
			color.Set(color.Reset)
			fmt.Print(": ")
			fmt.Println(text)
		}
	}
}

func getConfig() (string, map[string]string, error) {
	dir := os.Getenv("HOME")
	if dir == "" && runtime.GOOS == "windows" {
		dir = os.Getenv("APPDATA")
		if dir == "" {
			dir = filepath.Join(os.Getenv("USERPROFILE"), "Application Data", "twty")
		}
		dir = filepath.Join(dir, "twty")
	} else {
		dir = filepath.Join(dir, ".config", "twty")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", nil, err
	}
	var file string
	if *account == "" {
		file = filepath.Join(dir, "settings.json")
	} else {
		file = filepath.Join(dir, "settings-"+*account+".json")
	}
	config := map[string]string{}

	b, err := ioutil.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return "", nil, err
	}
	if err != nil {
		config["ClientToken"] = "MbartJkKCrSegn45xK9XLw"
		config["ClientSecret"] = "1nI3dHFtK9UY1kL6UEYWk6r2lFEcNHWhk7MtXe7eo"
	} else {
		err = json.Unmarshal(b, &config)
		if err != nil {
			return "", nil, fmt.Errorf("could not unmarshal %v: %v", file, err)
		}
	}
	return file, config, nil
}

var (
	account  = flag.String("a", "", "account")
	reply    = flag.Bool("r", false, "show replies")
	list     = flag.String("l", "", "show tweets")
	asjson   = flag.Bool("json", false, "show tweets as json")
	user     = flag.String("u", "", "show user timeline")
	favorite = flag.String("f", "", "specify favorite ID")
	search   = flag.String("s", "", "search word")
	stream   = flag.Bool("S", false, "stream timeline")
	inreply  = flag.String("i", "", "specify in-reply ID, if not specify text, it will be RT.")
	verbose  = flag.Bool("v", false, "detail display")
	debug    = flag.Bool("debug", false, "debug json")
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
  -json: as JSON
  -S: stream timeline
  -r: show replies
  -v: detail display
`)
	}
	flag.Parse()

	file, config, err := getConfig()
	if err != nil {
		log.Fatal("failed to get configuration:", err)
	}
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
		res := struct {
			Statuses       []Tweet `statuses`
			SearchMetadata `json:"search_metadata"`
		}{}
		err := rawCall(token, "GET", "https://api.twitter.com/1.1/search/tweets.json", map[string]string{"q": *search}, &res)
		if err != nil {
			log.Fatal("failed to get statuses:", err)
		}
		showTweets(res.Statuses, *verbose)
	} else if *reply {
		var tweets []Tweet
		err := rawCall(token, "GET", "https://api.twitter.com/1.1/statuses/mentions_timeline.json", map[string]string{}, &tweets)
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		showTweets(tweets, *verbose)
	} else if len(*list) > 0 {
		part := strings.SplitN(*list, "/", 2)
		if len(part) == 1 {
			var account Account
			err := rawCall(token, "GET", "https://api.twitter.com/1.1/account/settings.json", nil, &account)
			if err != nil {
				log.Fatal("failed to get account:", err)
			}
			part = []string{account.ScreenName, part[0]}
		}
		var tweets []Tweet
		err := rawCall(token, "GET", "https://api.twitter.com/1.1/lists/statuses.json", map[string]string{"owner_screen_name": part[0], "slug": part[1]}, &tweets)
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		showTweets(tweets, *verbose)
	} else if len(*user) > 0 {
		var tweets []Tweet
		err := rawCall(token, "GET", "https://api.twitter.com/1.1/statuses/user_timeline.json", map[string]string{"screen_name": *user}, &tweets)
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		showTweets(tweets, *verbose)
	} else if len(*favorite) > 0 {
		err := rawCall(token, "POST", "https://api.twitter.com/1.1/favorites/create.json", map[string]string{"id": *favorite}, nil)
		if err != nil {
			log.Fatal("failed to create favorite:", err)
		}
		fmt.Println("favorited")
	} else if *stream {
		url_ := "https://userstream.twitter.com/1.1/user.json"
		param := make(url.Values)
		oauthClient.SignParam(token, "GET", url_, param)
		url_ = url_ + "?" + param.Encode()
		resp, err := http.Get(url_)
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Fatal("failed to get tweets:", err)
		}
		var buf *bufio.Reader
		if resp.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(resp.Body)
			if err != nil {
				log.Fatal("failed to make gzip decoder:", err)
			}
			buf = bufio.NewReader(gr)
		} else {
			buf = bufio.NewReader(resp.Body)
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
			var tweet Tweet
			err := rawCall(token, "POST", "https://api.twitter.com/1.1/statuses/retweet/"+*inreply+".json", map[string]string{}, &tweet)
			if err != nil {
				log.Fatal("failed to retweet:", err)
			}
			fmt.Println("retweeted:", tweet.Identifier)
		} else {
			var tweets []Tweet
			err := rawCall(token, "GET", "https://api.twitter.com/1.1/statuses/home_timeline.json", map[string]string{}, &tweets)
			if err != nil {
				log.Fatal("failed to get tweets:", err)
			}
			showTweets(tweets, *verbose)
		}
	} else {
		var tweet Tweet
		err = rawCall(token, "POST", "https://api.twitter.com/1.1/statuses/update.json", map[string]string{"status": strings.Join(flag.Args(), " "), "in_reply_to_status_id": *inreply}, &tweet)
		if err != nil {
			log.Fatal("failed to post tweet:", err)
		}
		fmt.Println("tweeted:", tweet.Identifier)
	}
}
