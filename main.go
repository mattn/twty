package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"mime/multipart"
	"net"
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
)

const name = "twty"

const version = "0.1.1"

var revision = "HEAD"

const (
	_EmojiRedHeart    = "\u2764"
	_EmojiHighVoltage = "\u26A1"
)

const (
	defaultClientID     = "c3ZZTXhkc3lYMFdKYnpKSFNmeDE6MTpjaQ"
	defaultClientSecret = "e2XtHfI0BgavxOtEjLR2cstjFWI3p2ygq01A60fHJuPOczj8vW"
	authorizationURL    = "https://twitter.com/i/oauth2/authorize"
	tokenURL            = "https://api.twitter.com/2/oauth2/token"
	callbackPort        = 8989
	oauthScopes         = "tweet.read tweet.write users.read like.read like.write list.read offline.access"
)

type V2Tweet struct {
	ID               string              `json:"id"`
	Text             string              `json:"text"`
	AuthorID         string              `json:"author_id"`
	CreatedAt        string              `json:"created_at"`
	ReferencedTweets []V2ReferencedTweet `json:"referenced_tweets,omitempty"`
}

type V2ReferencedTweet struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type V2User struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url"`
}

type V2Includes struct {
	Users  []V2User  `json:"users"`
	Tweets []V2Tweet `json:"tweets"`
}

type V2Meta struct {
	ResultCount int    `json:"result_count"`
	NextToken   string `json:"next_token"`
	NewestID    string `json:"newest_id"`
	OldestID    string `json:"oldest_id"`
}

type V2TweetsResponse struct {
	Data     []V2Tweet  `json:"data"`
	Includes V2Includes `json:"includes"`
	Meta     V2Meta     `json:"meta"`
}

type V2TweetResponse struct {
	Data struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"data"`
}

type V2MeResponse struct {
	Data V2User `json:"data"`
}

type V2UserResponse struct {
	Data V2User `json:"data"`
}

type V2ListsResponse struct {
	Data []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Config struct {
	ClientID     string      `json:"client_id"`
	ClientSecret string      `json:"client_secret"`
	Token        OAuth2Token `json:"token"`
}

type files []string

func (f *files) String() string {
	return strings.Join([]string(*f), ",")
}

func (f *files) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func openBrowser(url string) {
	var browser string
	var args []string
	switch runtime.GOOS {
	case "windows":
		browser = "rundll32.exe"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		browser = "open"
		args = []string{url}
	case "plan9":
		browser = "plumb"
		args = []string{url}
	default:
		browser = "xdg-open"
		args = []string{url}
	}
	browser, err := exec.LookPath(browser)
	if err != nil {
		return
	}
	cmd := exec.Command(browser, args...)
	cmd.Stderr = os.Stderr
	cmd.Start()
}

func (app *App) authorize() error {
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)
	state := generateState()

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", callbackPort)

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {app.config.ClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {oauthScopes},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	authURL := authorizationURL + "?" + params.Encode()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", callbackPort))
	if err != nil {
		return fmt.Errorf("cannot start callback server on port %d: %v", callbackPort, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- errors.New("state mismatch")
			fmt.Fprint(w, "Error: state mismatch")
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errCh <- fmt.Errorf("authorization error: %s: %s", errMsg, r.URL.Query().Get("error_description"))
			fmt.Fprintf(w, "Error: %s", errMsg)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- errors.New("no code in callback")
			fmt.Fprint(w, "Error: no code received")
			return
		}
		codeCh <- code
		fmt.Fprint(w, "<html><body><p>Authorization successful! You can close this window.</p></body></html>")
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Shutdown(context.Background())

	color.Set(color.FgHiRed)
	fmt.Println("Open this URL to authorize.")
	color.Set(color.Reset)
	fmt.Println(authURL)
	openBrowser(authURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return errors.New("timeout waiting for authorization")
	}

	return app.exchangeCode(code, codeVerifier, redirectURI)
}

func (app *App) exchangeCode(code, codeVerifier, redirectURI string) error {
	data := url.Values{
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(app.config.ClientID, app.config.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token exchange failed: %s: %s", resp.Status, string(body))
	}

	return app.decodeTokenResponse(resp.Body)
}

func (app *App) refreshToken() error {
	data := url.Values{
		"refresh_token": {app.config.Token.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(app.config.ClientID, app.config.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: %s: %s", resp.Status, string(body))
	}

	return app.decodeTokenResponse(resp.Body)
}

func (app *App) decodeTokenResponse(r io.Reader) error {
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(r).Decode(&tokenResp); err != nil {
		return err
	}

	app.config.Token = OAuth2Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	return nil
}

func (app *App) ensureValidToken() error {
	if time.Now().Before(app.config.Token.ExpiresAt.Add(-30 * time.Second)) {
		return nil
	}
	if app.config.Token.RefreshToken == "" {
		return errors.New("token expired and no refresh token available, please re-authorize")
	}
	if err := app.refreshToken(); err != nil {
		return fmt.Errorf("cannot refresh token: %v", err)
	}
	return app.saveConfig()
}

func configDir() (string, error) {
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
		return "", err
	}
	return dir, nil
}

func (app *App) loadConfig() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if app.profile == "" {
		app.configFile = filepath.Join(dir, "settings.json")
	} else if app.profile == "?" {
		names, err := filepath.Glob(filepath.Join(dir, "settings*.json"))
		if err != nil {
			return err
		}
		for _, n := range names {
			n = filepath.Base(n)
			n = strings.TrimLeft(n[8:len(n)-5], "-")
			fmt.Println(n)
		}
		os.Exit(0)
	} else {
		app.configFile = filepath.Join(dir, "settings-"+app.profile+".json")
	}

	b, err := os.ReadFile(app.configFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err != nil {
		app.config = Config{
			ClientID:     defaultClientID,
			ClientSecret: defaultClientSecret,
		}
		return nil
	}

	return json.Unmarshal(b, &app.config)
}

func (app *App) saveConfig() error {
	b, err := json.MarshalIndent(app.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(app.configFile, b, 0600)
}

func (app *App) authorization() {
	if err := app.loadConfig(); err != nil {
		log.Fatalf("cannot load configuration: %v", err)
	}

	if app.config.Token.AccessToken == "" {
		if err := app.authorize(); err != nil {
			log.Fatalf("cannot authorize: %v", err)
		}
		if err := app.saveConfig(); err != nil {
			log.Fatalf("cannot save configuration: %v", err)
		}
	}
}

func (app *App) callGet(uri string, params map[string]string, res interface{}) error {
	if err := app.ensureValidToken(); err != nil {
		return err
	}

	reqURL := uri
	if len(params) > 0 {
		param := make(url.Values)
		for k, v := range params {
			param.Set(k, v)
		}
		reqURL = uri + "?" + param.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+app.config.Token.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	if res == nil {
		return nil
	}
	if app.debug {
		return json.NewDecoder(io.TeeReader(resp.Body, os.Stdout)).Decode(&res)
	}
	return json.NewDecoder(resp.Body).Decode(&res)
}

func (app *App) callPost(uri string, body interface{}, res interface{}) error {
	if err := app.ensureValidToken(); err != nil {
		return err
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+app.config.Token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	if res == nil {
		return nil
	}
	if app.debug {
		return json.NewDecoder(io.TeeReader(resp.Body, os.Stdout)).Decode(&res)
	}
	return json.NewDecoder(resp.Body).Decode(&res)
}

func (app *App) callPostForm(uri string, param url.Values, res interface{}) error {
	if err := app.ensureValidToken(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, uri, strings.NewReader(param.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+app.config.Token.AccessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	if res == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(&res)
}

func (app *App) callPostMultipart(uri string, buf *bytes.Buffer, contentType string, res interface{}) error {
	if err := app.ensureValidToken(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, uri, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+app.config.Token.AccessToken)
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	if res == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(&res)
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
	return http.DetectContentType(buf[:n]), nil
}

func (app *App) upload(file string) (string, error) {
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

	// INIT
	initRes := struct {
		MediaIDString string `json:"media_id_string"`
	}{}
	err = app.callPostForm(uri, url.Values{
		"command":     {"INIT"},
		"total_bytes": {fmt.Sprint(size)},
		"media_type":  {mediaType},
	}, &initRes)
	if err != nil {
		return "", fmt.Errorf("media upload INIT failed: %v", err)
	}

	// APPEND
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
		w := multipart.NewWriter(&buf)

		ww, err := w.CreateFormField("command")
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
		fmt.Fprint(ww, index)

		w.Close()

		err = app.callPostMultipart(uri, &buf, w.FormDataContentType(), nil)
		if err != nil {
			return "", fmt.Errorf("media upload APPEND failed: %v", err)
		}
		index++
		size -= int64(n)
	}

	// FINALIZE
	finalizeRes := struct {
		MediaIDString string `json:"media_id_string"`
	}{}
	err = app.callPostForm(uri, url.Values{
		"command":  {"FINALIZE"},
		"media_id": {initRes.MediaIDString},
	}, &finalizeRes)
	if err != nil {
		return "", fmt.Errorf("media upload FINALIZE failed: %v", err)
	}

	return finalizeRes.MediaIDString, nil
}

var replacer = strings.NewReplacer(
	"\r", "",
	"\n", " ",
	"\t", " ",
)

func showV2Tweets(res V2TweetsResponse, asjson bool, verbose bool) {
	if len(res.Data) == 0 {
		return
	}

	userMap := make(map[string]V2User)
	for _, u := range res.Includes.Users {
		userMap[u.ID] = u
	}
	tweetMap := make(map[string]V2Tweet)
	for _, t := range res.Includes.Tweets {
		tweetMap[t.ID] = t
	}

	if asjson {
		for _, tweet := range res.Data {
			json.NewEncoder(os.Stdout).Encode(tweet)
			os.Stdout.Sync()
		}
	} else if verbose {
		for i := len(res.Data) - 1; i >= 0; i-- {
			tweet := res.Data[i]
			user := userMap[tweet.AuthorID]
			text := tweetText(tweet, tweetMap)
			color.Set(color.FgHiRed)
			fmt.Println(user.Username + ": " + user.Name)
			color.Set(color.Reset)
			fmt.Println("  " + html.UnescapeString(text))
			fmt.Println("  " + tweet.ID)
			fmt.Println("  " + tweet.CreatedAt)
			fmt.Println()
		}
	} else {
		for i := len(res.Data) - 1; i >= 0; i-- {
			tweet := res.Data[i]
			user := userMap[tweet.AuthorID]
			text := tweetText(tweet, tweetMap)
			color.Set(color.FgHiRed)
			fmt.Print(user.Username)
			color.Set(color.Reset)
			fmt.Print(": ")
			fmt.Println(html.UnescapeString(text))
		}
	}
}

func tweetText(tweet V2Tweet, tweetMap map[string]V2Tweet) string {
	for _, ref := range tweet.ReferencedTweets {
		if ref.Type == "retweeted" {
			if rt, ok := tweetMap[ref.ID]; ok {
				return "RT: " + rt.Text
			}
		}
	}
	return tweet.Text
}

func v2TweetFields() map[string]string {
	return map[string]string{
		"tweet.fields": "created_at,author_id,text,referenced_tweets",
		"user.fields":  "name,username,profile_image_url",
		"expansions":   "author_id,referenced_tweets.id",
	}
}

func (app *App) getMyID() (string, error) {
	if app.myID != "" {
		return app.myID, nil
	}
	var res V2MeResponse
	err := app.callGet("https://api.twitter.com/2/users/me", nil, &res)
	if err != nil {
		return "", err
	}
	app.myID = res.Data.ID
	return app.myID, nil
}

func (app *App) searchTweets() {
	params := v2TweetFields()
	params["query"] = app.search
	if app.count != "" {
		params["max_results"] = app.count
	}
	if app.since != "" && isTimeFormat(app.since) {
		params["start_time"] = app.since + "T00:00:00Z"
	}
	if app.until != "" && isTimeFormat(app.until) {
		params["end_time"] = app.until + "T00:00:00Z"
	}
	if app.sinceID > 0 {
		params["since_id"] = strconv.FormatInt(app.sinceID, 10)
	}
	for {
		var res V2TweetsResponse
		err := app.callGet("https://api.twitter.com/2/tweets/search/recent", params, &res)
		if err != nil {
			log.Fatalf("cannot search tweets: %v", err)
		}
		if len(res.Data) > 0 {
			showV2Tweets(res, app.asjson, app.verbose)
		}
		if app.delay == 0 {
			break
		}
		if res.Meta.NewestID != "" {
			params["since_id"] = res.Meta.NewestID
		}
		time.Sleep(app.delay)
	}
}

func (app *App) showReplies() {
	myID, err := app.getMyID()
	if err != nil {
		log.Fatalf("cannot get user: %v", err)
	}

	params := v2TweetFields()
	if app.count != "" {
		params["max_results"] = app.count
	}

	var res V2TweetsResponse
	err = app.callGet("https://api.twitter.com/2/users/"+myID+"/mentions", params, &res)
	if err != nil {
		log.Fatalf("cannot get mentions: %v", err)
	}
	showV2Tweets(res, app.asjson, app.verbose)
}

func (app *App) showListTweets() {
	listID := app.list

	if _, err := strconv.ParseInt(listID, 10, 64); err != nil {
		part := strings.SplitN(app.list, "/", 2)

		var ownerID string
		if len(part) == 1 {
			id, err := app.getMyID()
			if err != nil {
				log.Fatalf("cannot get user: %v", err)
			}
			ownerID = id
		} else {
			var userRes V2UserResponse
			err := app.callGet("https://api.twitter.com/2/users/by/username/"+part[0], nil, &userRes)
			if err != nil {
				log.Fatalf("cannot look up user: %v", err)
			}
			ownerID = userRes.Data.ID
		}

		slug := part[len(part)-1]

		var listsRes V2ListsResponse
		err := app.callGet("https://api.twitter.com/2/users/"+ownerID+"/owned_lists", map[string]string{
			"list.fields": "name",
		}, &listsRes)
		if err != nil {
			log.Fatalf("cannot get lists: %v", err)
		}

		found := false
		for _, l := range listsRes.Data {
			if strings.EqualFold(l.Name, slug) {
				listID = l.ID
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("list not found: %s", slug)
		}
	}

	params := v2TweetFields()
	if app.count != "" {
		params["max_results"] = app.count
	}

	var res V2TweetsResponse
	err := app.callGet("https://api.twitter.com/2/lists/"+listID+"/tweets", params, &res)
	if err != nil {
		log.Fatalf("cannot get list tweets: %v", err)
	}
	showV2Tweets(res, app.asjson, app.verbose)
}

func (app *App) showUserTweets() {
	var userRes V2UserResponse
	err := app.callGet("https://api.twitter.com/2/users/by/username/"+app.user, nil, &userRes)
	if err != nil {
		log.Fatalf("cannot look up user: %v", err)
	}

	params := v2TweetFields()
	if app.count != "" {
		params["max_results"] = app.count
	}
	if app.sinceID > 0 {
		params["since_id"] = strconv.FormatInt(app.sinceID, 10)
	}
	if app.maxID > 0 {
		params["until_id"] = strconv.FormatInt(app.maxID, 10)
	}

	var res V2TweetsResponse
	err = app.callGet("https://api.twitter.com/2/users/"+userRes.Data.ID+"/tweets", params, &res)
	if err != nil {
		log.Fatalf("cannot get tweets: %v", err)
	}
	showV2Tweets(res, app.asjson, app.verbose)
}

func (app *App) favoriteTweet() {
	myID, err := app.getMyID()
	if err != nil {
		log.Fatalf("cannot get user: %v", err)
	}

	body := map[string]string{
		"tweet_id": app.favorite,
	}
	err = app.callPost("https://api.twitter.com/2/users/"+myID+"/likes", body, nil)
	if err != nil {
		log.Fatalf("cannot create favorite: %v", err)
	}
	color.Set(color.FgHiRed)
	fmt.Print(_EmojiRedHeart)
	color.Set(color.Reset)
	fmt.Println("favorited")
}

func (app *App) fromFile() {
	text, err := readFile(app.fromfile)
	if err != nil {
		log.Fatalf("cannot read a new tweet: %v", err)
	}
	body := map[string]interface{}{
		"text": string(text),
	}
	if app.inreply != "" {
		body["reply"] = map[string]string{
			"in_reply_to_tweet_id": app.inreply,
		}
	}
	if len(app.media) > 0 {
		body["media"] = map[string]interface{}{
			"media_ids": []string(app.media),
		}
	}
	var res V2TweetResponse
	err = app.callPost("https://api.twitter.com/2/tweets", body, &res)
	if err != nil {
		log.Fatalf("cannot post tweet: %v", err)
	}
	fmt.Println("tweeted:", res.Data.ID)
}

func (app *App) doRetweet() {
	myID, err := app.getMyID()
	if err != nil {
		log.Fatalf("cannot get user: %v", err)
	}

	body := map[string]string{
		"tweet_id": app.inreply,
	}
	err = app.callPost("https://api.twitter.com/2/users/"+myID+"/retweets", body, nil)
	if err != nil {
		log.Fatalf("cannot retweet: %v", err)
	}
	color.Set(color.FgHiYellow)
	fmt.Print(_EmojiHighVoltage)
	color.Set(color.Reset)
	fmt.Println("retweeted")
}

func (app *App) doStream() {
	myID, err := app.getMyID()
	if err != nil {
		log.Fatalf("cannot get user: %v", err)
	}

	params := v2TweetFields()
	var sinceID string
	for {
		if sinceID != "" {
			params["since_id"] = sinceID
		}
		var res V2TweetsResponse
		err := app.callGet("https://api.twitter.com/2/users/"+myID+"/timelines/reverse_chronological", params, &res)
		if err != nil {
			log.Fatalf("cannot get tweets: %v", err)
		}
		if len(res.Data) > 0 {
			showV2Tweets(res, app.asjson, app.verbose)
			sinceID = res.Meta.NewestID
		}
		time.Sleep(app.delay)
	}
}

func (app *App) doShow() {
	myID, err := app.getMyID()
	if err != nil {
		log.Fatalf("cannot get user: %v", err)
	}

	params := v2TweetFields()
	if app.count != "" {
		params["max_results"] = app.count
	}
	if app.sinceID > 0 {
		params["since_id"] = strconv.FormatInt(app.sinceID, 10)
	}
	if app.maxID > 0 {
		params["until_id"] = strconv.FormatInt(app.maxID, 10)
	}

	var res V2TweetsResponse
	err = app.callGet("https://api.twitter.com/2/users/"+myID+"/timelines/reverse_chronological", params, &res)
	if err != nil {
		log.Fatalf("cannot get tweets: %v", err)
	}
	showV2Tweets(res, app.asjson, app.verbose)
}

func (app *App) doTweet() {
	body := map[string]interface{}{
		"text": strings.Join(flag.Args(), " "),
	}
	if app.inreply != "" {
		body["reply"] = map[string]string{
			"in_reply_to_tweet_id": app.inreply,
		}
	}
	if len(app.media) > 0 {
		body["media"] = map[string]interface{}{
			"media_ids": []string(app.media),
		}
	}
	var res V2TweetResponse
	err := app.callPost("https://api.twitter.com/2/tweets", body, &res)
	if err != nil {
		log.Fatalf("cannot post tweet: %v", err)
	}
	fmt.Println("tweeted:", res.Data.ID)
}

type App struct {
	profile  string
	reply    bool
	list     string
	asjson   bool
	user     string
	favorite string
	search   string
	inreply  string
	delay    time.Duration
	media    files

	fromfile string
	count    string
	since    string
	until    string
	sinceID  int64
	maxID    int64

	config     Config
	configFile string
	myID       string

	verbose     bool
	showVersion bool
	debug       bool
}

func readFile(filename string) ([]byte, error) {
	if filename == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(filename)
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

func (app *App) uploadMedias() {
	var err error
	for i := range app.media {
		app.media[i], err = app.upload(app.media[i])
		if err != nil {
			log.Fatalf("cannot upload media: %v", err)
		}
	}
}

func parseFlags(app *App) {
	flag.StringVar(&app.profile, "a", os.Getenv("TWTY_ACCOUNT"), "account")
	flag.BoolVar(&app.reply, "r", false, "show replies")
	flag.StringVar(&app.list, "l", "", "show tweets")
	flag.BoolVar(&app.asjson, "json", false, "show tweets as json")
	flag.StringVar(&app.user, "u", "", "show user timeline")
	flag.StringVar(&app.favorite, "f", "", "specify favorite ID")
	flag.StringVar(&app.search, "s", "", "search word")
	flag.StringVar(&app.inreply, "i", "", "specify in-reply ID, if not specify text, it will be RT.")
	flag.Var(&app.media, "m", "upload media")
	flag.DurationVar(&app.delay, "S", 0, "delay")
	flag.BoolVar(&app.verbose, "v", false, "detail display")
	flag.BoolVar(&app.debug, "debug", false, "debug json")
	flag.BoolVar(&app.showVersion, "V", false, "Print the version")

	flag.StringVar(&app.fromfile, "ff", "", "post utf-8 string from a file(\"-\" means STDIN)")
	flag.StringVar(&app.count, "count", "", "fetch tweets count")
	flag.StringVar(&app.since, "since", "", "fetch tweets since date.")
	flag.StringVar(&app.until, "until", "", "fetch tweets until date.")
	flag.Int64Var(&app.sinceID, "since_id", 0, "fetch tweets that id is greater than since_id.")
	flag.Int64Var(&app.maxID, "max_id", 0, "fetch tweets that id is lower than max_id.")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()
}

const usage = `Usage of twty:
  -a PROFILE: switch profile to load configuration file.
  -f ID: specify favorite ID
  -i ID: specify in-reply ID, if not specify text, it will be RT.
  -l LIST: show list's timeline (list ID or user/list-name)
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
`

func main() {
	var app App

	parseFlags(&app)
	if app.showVersion {
		fmt.Printf("%s %s (rev: %s/%s)\n", name, version, revision, runtime.Version())
		return
	}

	app.authorization()

	if len(app.media) > 0 {
		app.uploadMedias()
	}

	if len(app.search) > 0 {
		app.searchTweets()
	} else if app.reply {
		app.showReplies()
	} else if app.list != "" {
		app.showListTweets()
	} else if app.user != "" {
		app.showUserTweets()
	} else if app.favorite != "" {
		app.favoriteTweet()
	} else if app.fromfile != "" {
		app.fromFile()
	} else if flag.NArg() == 0 && len(app.media) == 0 {
		if app.inreply != "" {
			app.doRetweet()
		} else if app.delay > 0 {
			app.doStream()
		} else {
			app.doShow()
		}
	} else {
		app.doTweet()
	}
}
