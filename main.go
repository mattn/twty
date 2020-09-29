package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/garyburd/go-oauth/oauth"
)

const name = "twty"

const version = "0.0.9"

var revision = "HEAD"

const (
	_EmojiRedHeart    = "\u2764"
	_EmojiHighVoltage = "\u26A1"
)

// Account hold information about account
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

// Tweet hold information about tweet
type Tweet struct {
	Text       string `json:"text"`
	FullText   string `json:"full_text,omitempty"`
	Identifier string `json:"id_str"`
	Source     string `json:"source"`
	CreatedAt  string `json:"created_at"`
	User       struct {
		Name            string `json:"name"`
		ScreenName      string `json:"screen_name"`
		FollowersCount  int    `json:"followers_count"`
		ProfileImageURL string `json:"profile_image_url"`
	} `json:"user"`
	RetweetedStatus *struct {
		FullText string `json:"full_text"`
	} `json:"retweeted_status"`
	Place *struct {
		ID       string `json:"id"`
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
			URL     string `json:"url"`
		} `json:"urls"`
	} `json:"entities"`
}

// SearchMetadata hold information about search metadata
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

// RSS hold information about RSS
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
			GUID        string
			Author      string
		}
	}
}

type files []string

func (f *files) String() string {
	return strings.Join([]string(*f), ",")
}

func (f *files) Set(value string) error {
	*f = append(*f, value)
	return nil
}

var oauthClient = oauth.Client{
	TemporaryCredentialRequestURI: "https://api.twitter.com/oauth/request_token",
	ResourceOwnerAuthorizationURI: "https://api.twitter.com/oauth/authenticate",
	TokenRequestURI:               "https://api.twitter.com/oauth/access_token",
}

func makeopt(v ...string) map[string]string {
	opt := map[string]string{}
	for i := 0; i < len(v); i += 2 {
		opt[v[i]] = v[i+1]
	}
	return opt
}

func clientAuth(requestToken *oauth.Credentials) (*oauth.Credentials, error) {
	var err error
	browser := "xdg-open"
	url := oauthClient.AuthorizationURL(requestToken, nil)

	args := []string{url}
	if runtime.GOOS == "windows" {
		browser = "rundll32.exe"
		args = []string{"url.dll,FileProtocolHandler", url}
	} else if runtime.GOOS == "darwin" {
		browser = "open"
		args = []string{url}
	} else if runtime.GOOS == "plan9" {
		browser = "plumb"
	}
	color.Set(color.FgHiRed)
	fmt.Println("Open this URL and enter PIN.")
	color.Set(color.Reset)
	fmt.Println(url)
	browser, err = exec.LookPath(browser)
	if err == nil {
		cmd := exec.Command(browser, args...)
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			return nil, fmt.Errorf("cannot start command: %v", err)
		}
	}

	fmt.Print("PIN: ")
	stdin := bufio.NewScanner(os.Stdin)
	if !stdin.Scan() {
		return nil, fmt.Errorf("canceled")
	}
	accessToken, _, err := oauthClient.RequestToken(http.DefaultClient, requestToken, stdin.Text())
	if err != nil {
		return nil, fmt.Errorf("cannot request token: %v", err)
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
		token = &oauth.Credentials{Token: accessToken, Secret: accessSecret}
	} else {
		requestToken, err := oauthClient.RequestTemporaryCredentials(http.DefaultClient, "", nil)
		if err != nil {
			err = fmt.Errorf("cannot request temporary credentials: %v", err)
			return nil, false, err
		}
		token, err = clientAuth(requestToken)
		if err != nil {
			err = fmt.Errorf("cannot request temporary credentials: %v", err)
			return nil, false, err
		}

		config["AccessToken"] = token.Token
		config["AccessSecret"] = token.Secret
		authorized = true
	}
	return token, authorized, nil
}

func contentTypeOf(file string) (string, error) {
	buf := make([]byte, 512)
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	n, err := f.Read(buf)
	if err != nil {
		return "", err
	}
	ct := http.DetectContentType(buf[:n])
	return ct, nil
}

func upload(token *oauth.Credentials, file string, opt map[string]string) (string, error) {
	mediaType, _ := contentTypeOf(file)
	if mediaType == "" {
		ext := filepath.Ext(strings.ToLower(file))
		switch ext {
		case ".jpg", ".jpeg":
			mediaType = "image/jpeg"
		case ".png":
			mediaType = "image/png"
		case ".mp4":
			mediaType = "video/mp4"
		case ".gif":
			mediaType = "image/gif"
		default:
			return "", errors.New("unrecognized media type")
		}
	}
	ft, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	size := ft.Size()

	uri := "https://upload.twitter.com/1.1/media/upload.json"
	param := make(url.Values)
	for k, v := range opt {
		param.Set(k, v)
	}
	param.Set("command", "INIT")
	param.Set("total_bytes", fmt.Sprint(size))
	param.Set("media_type", mediaType)

	req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(param.Encode()))
	if err != nil {
		return "", err
	}

	oauthClient.SignParam(token, http.MethodPost, uri, param)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "OAuth "+strings.Replace(param.Encode(), "&", ",", -1))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	initRes := struct {
		ExpiresAfterSecs int64 `json:"expires_after_secs"`
		Image            struct {
			H         int64  `json:"h"`
			ImageType string `json:"image_type"`
			W         int64  `json:"w"`
		} `json:"image"`
		MediaID       int64  `json:"media_id"`
		MediaIDString string `json:"media_id_string"`
		Size          int64  `json:"size"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&initRes)
	resp.Body.Close()
	if err != nil {
		return "", err
	}

	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	index := 0
	for size > 0 {
		var payload [1024 * 5000]byte
		n, err := f.Read(payload[:])
		if err != nil {
			return "", err
		}

		var buf bytes.Buffer
		var ww io.Writer

		w := multipart.NewWriter(&buf)

		ww, err = w.CreateFormField("command")
		if err != nil {
			return "", err
		}
		fmt.Fprint(ww, "APPEND")

		ww, err = w.CreateFormField("media_id")
		if err != nil {
			return "", err
		}
		fmt.Fprint(ww, initRes.MediaIDString)

		ww, err = w.CreateFormField("media")
		if err != nil {
			return "", err
		}
		ww.Write(payload[:n])

		ww, err = w.CreateFormField("segment_index")
		if err != nil {
			return "", err
		}
		fmt.Fprint(ww, fmt.Sprint(index))

		w.Close()

		param = make(url.Values)
		for k, v := range opt {
			param.Set(k, v)
		}
		req, err := http.NewRequest(http.MethodPost, uri, &buf)
		if err != nil {
			return "", err
		}

		oauthClient.SignParam(token, http.MethodPost, uri, param)

		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "OAuth "+strings.Replace(param.Encode(), "&", ",", -1))

		_, err = http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		resp.Body.Close()
		index++
		size -= int64(n)
	}

	param = make(url.Values)
	for k, v := range opt {
		param.Set(k, v)
	}
	param.Set("command", "FINALIZE")
	param.Set("media_id", initRes.MediaIDString)

	req, err = http.NewRequest(http.MethodPost, uri, strings.NewReader(param.Encode()))
	if err != nil {
		return "", err
	}

	oauthClient.SignParam(token, http.MethodPost, uri, param)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "OAuth "+strings.Replace(param.Encode(), "&", ",", -1))

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	finalizeRes := struct {
		ExpiresAfterSecs int64  `json:"expires_after_secs"`
		MediaID          int64  `json:"media_id"`
		MediaIDString    string `json:"media_id_string"`
		Size             int64  `json:"size"`
		Video            struct {
			VideoType string `json:"video_type"`
		} `json:"video"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&finalizeRes)
	resp.Body.Close()
	if err != nil {
		return "", err
	}
	return finalizeRes.MediaIDString, nil
}

func rawCall(token *oauth.Credentials, method string, uri string, opt map[string]string, res interface{}) error {
	param := make(url.Values)
	for k, v := range opt {
		param.Set(k, v)
	}
	oauthClient.SignParam(token, method, uri, param)
	var resp *http.Response
	var err error
	if method == http.MethodGet {
		uri = uri + "?" + param.Encode()
		resp, err = http.Get(uri)
	} else {
		resp, err = http.PostForm(uri, url.Values(param))
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return errors.New(resp.Status)
	}
	if res == nil {
		return nil
	}
	if debug {
		return json.NewDecoder(io.TeeReader(resp.Body, os.Stdout)).Decode(&res)
	}
	return json.NewDecoder(resp.Body).Decode(&res)
}

var replacer = strings.NewReplacer(
	"\r", "",
	"\n", " ",
	"\t", " ",
)

const _TimeLayout = "Mon Jan 02 15:04:05 -0700 2006"

func toLocalTime(timeStr string) string {
	timeValue, err := time.Parse(_TimeLayout, timeStr)
	if err != nil {
		return timeStr
	}
	return timeValue.Local().Format(_TimeLayout)
}

func showTweets(tweets []Tweet, asjson bool, verbose bool) {
	if asjson {
		for _, tweet := range tweets {
			if tweet.RetweetedStatus != nil {
				tweet.Text = tweet.RetweetedStatus.FullText
			} else if tweet.FullText != "" {
				tweet.Text = tweet.FullText
				tweet.FullText = ""
			}
			json.NewEncoder(os.Stdout).Encode(tweet)
			os.Stdout.Sync()
		}
	} else if verbose {
		for i := len(tweets) - 1; i >= 0; i-- {
			name := tweets[i].User.Name
			user := tweets[i].User.ScreenName
			var text string
			if tweets[i].RetweetedStatus != nil {
				tweets[i].Text = tweets[i].RetweetedStatus.FullText
			} else if tweets[i].FullText != "" {
				text = tweets[i].FullText
			} else {
				text = tweets[i].Text
			}
			text = replacer.Replace(text)
			color.Set(color.FgHiRed)
			fmt.Println(user + ": " + name)
			color.Set(color.Reset)
			fmt.Println("  " + html.UnescapeString(text))
			fmt.Println("  " + tweets[i].Identifier)
			fmt.Println("  " + toLocalTime(tweets[i].CreatedAt))
			fmt.Println()
		}
	} else {
		for i := len(tweets) - 1; i >= 0; i-- {
			user := tweets[i].User.ScreenName
			var text string
			if tweets[i].RetweetedStatus != nil {
				text = tweets[i].RetweetedStatus.FullText
			} else if tweets[i].FullText != "" {
				text = tweets[i].FullText
			} else {
				text = tweets[i].Text
			}
			color.Set(color.FgHiRed)
			fmt.Print(user)
			color.Set(color.Reset)
			fmt.Print(": ")
			fmt.Println(html.UnescapeString(text))
		}
	}
}

func getConfig(profile string) (string, map[string]string, error) {
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
	if profile == "" {
		file = filepath.Join(dir, "settings.json")
	} else if profile == "?" {
		names, err := filepath.Glob(filepath.Join(dir, "settings*.json"))
		if err != nil {
			return "", nil, err
		}
		for _, name := range names {
			name = filepath.Base(name)
			name = strings.TrimLeft(name[8:len(name)-5], "-")
			fmt.Println(name)
		}
		os.Exit(0)
	} else {
		file = filepath.Join(dir, "settings-"+profile+".json")
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
	debug bool
)

func readFile(filename string) ([]byte, error) {
	if filename == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	return ioutil.ReadFile(filename)
}

func countToOpt(opt map[string]string, c string) map[string]string {
	if c != "" {
		_, err := strconv.Atoi(c)
		if err == nil {
			opt["count"] = c
		}
	}
	return opt
}

func sinceToOpt(opt map[string]string, timeFormat string) map[string]string {
	return timeFormatToOpt(opt, "since", timeFormat)
}

func untilToOpt(opt map[string]string, timeFormat string) map[string]string {
	return timeFormatToOpt(opt, "until", timeFormat)
}

func timeFormatToOpt(opt map[string]string, key string, timeFormat string) map[string]string {
	if timeFormat != "" || !isTimeFormat(timeFormat) {
		return opt
	}
	opt[key] = timeFormat
	return opt
}

func sinceIDtoOpt(opt map[string]string, id int64) map[string]string {
	return idToOpt(opt, "since_id", id)
}

func maxIDtoOpt(opt map[string]string, id int64) map[string]string {
	return idToOpt(opt, "max_id", id)
}

func idToOpt(opt map[string]string, key string, id int64) map[string]string {
	if id < 1 {
		return opt
	}
	opt[key] = strconv.FormatInt(id, 10)
	return opt
}

// isTimeFormat returns true if the parameter string matches the format like '[0-9]+-[0-9]+-[0-9]+'
func isTimeFormat(t string) bool {
	splitFormat := strings.Split(t, "-")
	if len(splitFormat) != 3 {
		return false
	}

	for _, v := range splitFormat {
		_, err := strconv.Atoi(v)
		if err != nil {
			return false
		}
	}

	return true
}

func main() {
	var profile string
	var reply bool
	var list string
	var asjson bool
	var user string
	var favorite string
	var search string
	var inreply string
	var delay time.Duration
	var media files
	var verbose bool
	var showVersion bool

	flag.StringVar(&profile, "a", os.Getenv("TWTY_ACCOUNT"), "account")
	flag.BoolVar(&reply, "r", false, "show replies")
	flag.StringVar(&list, "l", "", "show tweets")
	flag.BoolVar(&asjson, "json", false, "show tweets as json")
	flag.StringVar(&user, "u", "", "show user timeline")
	flag.StringVar(&favorite, "f", "", "specify favorite ID")
	flag.StringVar(&search, "s", "", "search word")
	flag.StringVar(&inreply, "i", "", "specify in-reply ID, if not specify text, it will be RT.")
	flag.Var(&media, "m", "upload media")
	flag.DurationVar(&delay, "S", 0, "delay")
	flag.BoolVar(&verbose, "v", false, "detail display")
	flag.BoolVar(&debug, "debug", false, "debug json")
	flag.BoolVar(&showVersion, "V", false, "Print the version")

	var fromfile string
	var count string
	var since string
	var until string
	var sinceID int64
	var maxID int64

	flag.StringVar(&fromfile, "ff", "", "post utf-8 string from a file(\"-\" means STDIN)")
	flag.StringVar(&count, "count", "", "fetch tweets count")
	flag.StringVar(&since, "since", "", "fetch tweets since date.")
	flag.StringVar(&until, "until", "", "fetch tweets until date.")
	flag.Int64Var(&sinceID, "since_id", 0, "fetch tweets that id is greater than since_id.")
	flag.Int64Var(&maxID, "max_id", 0, "fetch tweets that id is lower than max_id.")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage of twty:
  -a PROFILE: switch profile to load configuration file.
  -f ID: specify favorite ID
  -i ID: specify in-reply ID, if not specify text, it will be RT.
  -l USER/LIST: show list's timeline (ex: mattn_jp/subtech)
  -m FILE: upload media
  -u USER: show user's timeline
  -s WORD: search timeline
  -S DELAY tweets after DELAY
  -json: as JSON
  -r: show replies
  -v: detail display
  -ff FILENAME: post utf-8 string from a file("-" means STDIN)
  -count NUMBER: show NUMBER tweets at timeline.
  -since DATE: show tweets created after the DATE (ex. 2017-05-01)
  -until DATE: show tweets created before the DATE (ex. 2017-05-31)
  -since_id NUMBER: show tweets that have ids greater than NUMBER.
  -max_id NUMBER: show tweets that have ids lower than NUMBER.
`)
	}
	flag.Parse()

	if showVersion {
		fmt.Printf("%s %s (rev: %s/%s)\n", name, version, revision, runtime.Version())
		return
	}
	os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",http2client=0")

	file, config, err := getConfig(profile)
	if err != nil {
		log.Fatalf("cannot get configuration: %v", err)
	}
	token, authorized, err := getAccessToken(config)
	if err != nil {
		log.Fatalf("cannot get access token: %v", err)
	}
	if authorized {
		b, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			log.Fatalf("cannot store file: %v", err)
		}
		err = ioutil.WriteFile(file, b, 0700)
		if err != nil {
			log.Fatalf("cannot store file: %v", err)
		}
	}

	if len(media) > 0 {
		for i := range media {
			media[i], err = upload(token, media[i], nil)
			if err != nil {
				log.Fatalf("cannot upload media: %v", err)
			}
		}
	}

	if len(search) > 0 {
		res := struct {
			Statuses       []Tweet `json:"statuses"`
			SearchMetadata `json:"search_metadata"`
		}{}
		opt := makeopt(
			"tweet_mode", "extended",
			"q", search,
		)
		opt = countToOpt(opt, count)
		opt = sinceToOpt(opt, since)
		opt = untilToOpt(opt, until)
		err := rawCall(token, http.MethodGet, "https://api.twitter.com/1.1/search/tweets.json", opt, &res)
		if err != nil {
			log.Fatalf("cannot get statuses: %v", err)
		}
		showTweets(res.Statuses, asjson, verbose)
	} else if reply {
		var tweets []Tweet
		opt := makeopt(
			"tweet_mode", "extended",
		)
		opt = countToOpt(opt, count)
		err := rawCall(token, http.MethodGet, "https://api.twitter.com/1.1/statuses/mentions_timeline.json", opt, &tweets)
		if err != nil {
			log.Fatalf("cannot get tweets: %v", err)
		}
		showTweets(tweets, asjson, verbose)
	} else if list != "" {
		part := strings.SplitN(list, "/", 2)
		if len(part) == 1 {
			var account Account
			err := rawCall(token, http.MethodGet, "https://api.twitter.com/1.1/account/settings.json", nil, &account)
			if err != nil {
				log.Fatalf("cannot get account: %v", err)
			}
			part = []string{account.ScreenName, part[0]}
		}
		var tweets []Tweet
		opt := makeopt(
			"tweet_mode", "extended",
			"owner_screen_name", part[0],
			"slug", part[1],
		)
		opt = countToOpt(opt, count)
		opt = sinceIDtoOpt(opt, sinceID)
		opt = maxIDtoOpt(opt, maxID)
		err := rawCall(token, http.MethodGet, "https://api.twitter.com/1.1/lists/statuses.json", opt, &tweets)
		if err != nil {
			log.Fatalf("cannot get tweets: %v", err)
		}
		showTweets(tweets, asjson, verbose)
	} else if user != "" {
		var tweets []Tweet
		opt := makeopt(
			"tweet_mode", "extended",
			"screen_name", user,
		)
		opt = countToOpt(opt, count)
		opt = sinceIDtoOpt(opt, sinceID)
		opt = maxIDtoOpt(opt, maxID)
		err := rawCall(token, http.MethodGet, "https://api.twitter.com/1.1/statuses/user_timeline.json", opt, &tweets)
		if err != nil {
			log.Fatalf("cannot get tweets: %v", err)
		}
		showTweets(tweets, asjson, verbose)
	} else if favorite != "" {
		opt := makeopt(
			"id", favorite,
		)
		err := rawCall(token, http.MethodPost, "https://api.twitter.com/1.1/favorites/create.json", opt, nil)
		if err != nil {
			log.Fatalf("cannot create favorite: %v", err)
		}
		color.Set(color.FgHiRed)
		fmt.Print(_EmojiRedHeart)
		color.Set(color.Reset)
		fmt.Println("favorited")
	} else if fromfile != "" {
		text, err := readFile(fromfile)
		if err != nil {
			log.Fatalf("cannot read a new tweet: %v", err)
		}
		var tweet Tweet
		opt := makeopt(
			"status", string(text),
			"in_reply_to_status_id", inreply,
			"media_ids", media.String(),
		)
		err = rawCall(token, http.MethodPost, "https://api.twitter.com/1.1/statuses/update.json", opt, &tweet)
		if err != nil {
			log.Fatalf("cannot post tweet: %v", err)
		}
		fmt.Println("tweeted:", tweet.Identifier)
	} else if flag.NArg() == 0 && len(media) == 0 {
		if inreply != "" {
			var tweet Tweet
			opt := makeopt("tweet_mode", "extended")
			opt = countToOpt(opt, count)
			err := rawCall(token, http.MethodPost, "https://api.twitter.com/1.1/statuses/retweet/"+inreply+".json", opt, &tweet)
			if err != nil {
				log.Fatalf("cannot retweet: %v", err)
			}
			color.Set(color.FgHiYellow)
			fmt.Print(_EmojiHighVoltage)
			color.Set(color.Reset)
			fmt.Println("retweeted:", tweet.Identifier)
		} else if delay > 0 {
			var tweets []Tweet
			opt := makeopt()
			for {
				opt = sinceToOpt(opt, since)
				err := rawCall(token, http.MethodGet, "https://api.twitter.com/1.1/statuses/home_timeline.json", opt, &tweets)
				if err != nil {
					log.Fatalf("cannot get tweets: %v", err)
				}
				if len(tweets) > 0 {
					showTweets(tweets, asjson, verbose)
					since = tweets[len(tweets)-1].CreatedAt
				}
				time.Sleep(delay)
			}
		} else {
			var tweets []Tweet
			opt := makeopt("tweet_mode", "extended")
			opt = countToOpt(opt, count)
			err := rawCall(token, http.MethodGet, "https://api.twitter.com/1.1/statuses/home_timeline.json", opt, &tweets)
			if err != nil {
				log.Fatalf("cannot get tweets: %v", err)
			}
			showTweets(tweets, asjson, verbose)
		}
	} else {
		var tweet Tweet
		opt := makeopt(
			"status", strings.Join(flag.Args(), " "),
			"in_reply_to_status_id", inreply,
			"media_ids", media.String(),
		)
		err = rawCall(token, http.MethodPost, "https://api.twitter.com/1.1/statuses/update.json", opt, &tweet)
		if err != nil {
			log.Fatalf("cannot post tweet: %v", err)
		}
		fmt.Println("tweeted:", tweet.Identifier)
	}
}
