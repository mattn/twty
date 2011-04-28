package main

import (
	"bufio"
	"exec"
	"flag"
	"fmt"
	"github.com/garyburd/twister/oauth"
	"github.com/garyburd/twister/web"
	"http"
	"iconv"
	"io/ioutil"
	"json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"xml"
)

type Tweet struct {
	Text       string
	Identifier string "id_str"
	Source     string
	CreatedAt  string "created_at"
	User       struct {
		Name            string
		ScreenName      string "screen_name"
		FollowersCount  int    "followers_count"
		ProfileImageURL string "profile_image_url"
	}
	Place *struct {
		Id       string
		FullName string "full_name"
	}
	Entities struct {
		HashTags []struct {
			Indices [2]int
			Text    string
		}
		UserMentions []struct {
			Indices    [2]int
			ScreenName string "screen_name"
		}    "user_mentions"
		Urls []struct {
			Indices [2]int
			Url     string
		}
	}
}

type RSS struct {
	Channel struct {
		Title string
		Description string
		Link string
		Item []struct {
			Title string
			Description string
			PubDate string
			Link []string
			Guid string
			Author string
		}
	}
}

var oauthClient = oauth.Client{
	TemporaryCredentialRequestURI: "https://api.twitter.com/oauth/request_token",
	ResourceOwnerAuthorizationURI: "https://api.twitter.com/oauth/authenticate",
	TokenRequestURI:               "https://api.twitter.com/oauth/access_token",
}

func clientAuth(requestToken *oauth.Credentials) (*oauth.Credentials, os.Error) {
	cmd := "xdg-open"
	url := oauthClient.AuthorizationURL(requestToken)

	args := []string{cmd, url}
	if syscall.OS == "windows" {
		cmd = "rundll32.exe"
		args = []string{cmd, "url.dll,FileProtocolHandler", url}
	} else if syscall.OS == "darwin" {
		cmd = "open"
		args = []string{cmd, url}
	}
	cmd, err := exec.LookPath(cmd)
	if err != nil {
		log.Fatal("command not found:", err)
	}
	p, err := os.StartProcess(cmd, args, &os.ProcAttr{Dir: "", Files: []*os.File{nil, nil, os.Stderr}})
	if err != nil {
		log.Fatal("failed to start command:", err)
	}
	defer p.Release()

	print("PIN: ")
	stdin := bufio.NewReader(os.Stdin)
	b, err := stdin.ReadBytes('\n')
	if err != nil {
		log.Fatal("canceled")
	}

	accessToken, _, err := oauthClient.RequestToken(requestToken, string(b[0:len(b)-2]))
	if err != nil {
		log.Fatal("failed to request token:", err)
	}
	return accessToken, nil
}

func getAccessToken(config map[string]string) (*oauth.Credentials, bool, os.Error) {
	oauthClient.Credentials.Token = config["ClientToken"]
	oauthClient.Credentials.Secret = config["ClientSecret"]

	authorized := false
	var token *oauth.Credentials
	accessToken, foundToken := config["AccessToken"]
	accessSecert, foundSecret := config["AccessSecret"]
	if foundToken && foundSecret {
		token = &oauth.Credentials{accessToken, accessSecert}
	} else {
		requestToken, err := oauthClient.RequestTemporaryCredentials("")
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

func getTweets(token *oauth.Credentials, url string, opt map[string]string) ([]Tweet, os.Error) {
	param := make(web.ParamMap)
	for k, v := range opt {
		param.Set(k, v)
	}
	oauthClient.SignParam(token, "GET", url, param)
	url = url + "?" + param.FormEncodedString()
	res, _, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, err
	}
	var tweets []Tweet
	err = json.NewDecoder(res.Body).Decode(&tweets)
	if err != nil {
		return nil, err
	}
	return tweets, nil
}

func showRSS(rss RSS) {
	ic, err := iconv.Open("char", "UTF-8")
	if err != nil {
		log.Fatal("failed to convert string:", err)
	}
	defer ic.Close()
	items := rss.Channel.Item
	for i := len(items) - 1; i >= 0; i-- {
		user := strings.Split(items[i].Author, "@", 2)[0]
		user, _ = ic.Conv(user)
		text, _ := ic.Conv(items[i].Title)
		println(user + ": " + text)
	}
}

func showTweets(tweets []Tweet, verbose bool) {
	ic, err := iconv.Open("char", "UTF-8")
	if err != nil {
		log.Fatal("failed to convert string:", err)
	}
	defer ic.Close()
	if verbose {
		for i := len(tweets) - 1; i >= 0; i-- {
			name, _ := ic.Conv(tweets[i].User.Name)
			user, _ := ic.Conv(tweets[i].User.ScreenName)
			text := tweets[i].Text
			text = strings.Replace(text, "\r", "", -1)
			text = strings.Replace(text, "\n", " ", -1)
			text = strings.Replace(text, "\t", " ", -1)
			text, _ = ic.Conv(text)
			println(user + ": " + name)
			println("  " + text)
			println("  " + tweets[i].Identifier)
			println("  " + tweets[i].CreatedAt)
			println()
		}
	} else {
		for i := len(tweets) - 1; i >= 0; i-- {
			user, _ := ic.Conv(tweets[i].User.ScreenName)
			text, _ := ic.Conv(tweets[i].Text)
			println(user + ": " + text)
		}
	}
}

func postTweet(token *oauth.Credentials, url string, opt map[string]string) os.Error {
	param := make(web.ParamMap)
	for k, v := range opt {
		param.Set(k, v)
	}
	oauthClient.SignParam(token, "POST", url, param)
	res, err := http.PostForm(url, param.StringMap())
	if err != nil {
		log.Println("failed to post tweet:", err)
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Println("failed to get timeline:", err)
		return err
	}
	return nil
}

func getConfig() (string, map[string]string) {
	home := os.Getenv("HOME")
	dir := filepath.Join(home, ".config")
	if syscall.OS == "windows" {
		home = os.Getenv("USERPROFILE")
		dir = filepath.Join(home, "Application Data")
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
	file := filepath.Join(dir, "settings.json")
	config := map[string]string{}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		config["ClientToken"] = "MbartJkKCrSegn45xK9XLw"
		config["ClientSecret"] = "1nI3dHFtK9UY1kL6UEYWk6r2lFEcNHWhk7MtXe7eo"
	} else {
		err = json.Unmarshal(b, &config)
		if err != nil {
			log.Fatal("could not unmarhal settings.json:", err)
		}
	}
	return file, config
}

func main() {
	reply := flag.Bool("r", false, "show replies")
	list := flag.String("l", "", "show tweets")
	user := flag.String("u", "", "show user timeline")
	favorite := flag.String("f", "", "specify favorite ID")
	search := flag.String("s", "", "search word")
	inreply := flag.String("i", "", "specify in-reply ID, if not specify text, it will be RT.")
	verbose := flag.Bool("v", false, "detail display")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage of twty:
  -f ID: specify favorite ID
  -i ID: specify in-reply ID, if not specify text, it will be RT.
  -l USER/LIST: show list's timeline (ex: mattn_jp/subtech)
  -u USER: show user's timeline
  -r: show replies
  -v: detail display
`)
	}
	flag.Parse()

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
		res, _, err := http.Get("http://search.twitter.com/search.rss?q=" + http.URLEscape(*search))
		if err != nil {
			log.Fatal("failed to search word:", err)
		}
		defer res.Body.Close()
		var rss RSS
		err = xml.Unmarshal(res.Body, &rss)
		if err != nil {
			log.Fatal("could not unmarhal response:", err)
		}
		showRSS(rss)
	} else if *reply {
		tweets, err := getTweets(token, "https://api.twitter.com/1/statuses/mentions.json", map[string]string{})
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		showTweets(tweets, *verbose)
	} else if len(*list) > 0 {
		part := strings.Split(*list, "/", 2)
		tweets, err := getTweets(token, "https://api.twitter.com/1/"+part[0]+"/lists/"+part[1]+"/statuses.json", map[string]string{})
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		showTweets(tweets, *verbose)
	} else if len(*user) > 0 {
		tweets, err := getTweets(token, "https://api.twitter.com/1/statuses/user_timeline.json", map[string]string{"screen_name": *user})
		if err != nil {
			log.Fatal("failed to get tweets:", err)
		}
		showTweets(tweets, *verbose)
	} else if len(*favorite) > 0 {
		postTweet(token, "https://api.twitter.com/1/favorites/create/"+*favorite+".json", map[string]string{})
	} else if flag.NArg() == 0 {
		if len(*inreply) > 0 {
			postTweet(token, "https://api.twitter.com/1/statuses/retweet/"+*inreply+".json", map[string]string{})
		} else {
			tweets, err := getTweets(token, "https://api.twitter.com/1/statuses/home_timeline.json", map[string]string{})
			if err != nil {
				log.Fatal("failed to get tweets:", err)
			}
			showTweets(tweets, *verbose)
		}
	} else {
		postTweet(token, "https://api.twitter.com/1/statuses/update.json", map[string]string{"status": strings.Join(flag.Args(), " "), "in_reply_to_status_id": *inreply})
	}
}
