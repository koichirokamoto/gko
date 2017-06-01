package social

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/koichirokamoto/gko/util"
)

const (
	api            = "https://api.twitter.com"
	twitterAuthAPI = "https://api.twitter.com/oauth/authenticate?oauth_token=%s"
)

// TwitterToken is twitter token.
type TwitterToken struct {
	oauthToken       string
	oauthTokenSecret string
}

// TwitterClient is twitter authentication.
type TwitterClient struct {
	consumerKey      string
	consumerSecret   string
	oauthToken       string
	oauthTokenSecret string
	client           *http.Client
}

// NewTwitterClient returns new twitter authentication.
func NewTwitterClient(client *http.Client, consumerKey, consumerSecret, oauthToken, oauthTokenSecret string) *TwitterClient {
	return &TwitterClient{
		consumerKey:      consumerKey,
		consumerSecret:   consumerSecret,
		oauthToken:       oauthToken,
		oauthTokenSecret: oauthTokenSecret,
		client:           client,
	}
}

// LoginURL create twitter login url.
func (t *TwitterClient) LoginURL(callbackURL string) (string, error) {
	params := url.Values{}
	params.Set("oauth_callback", callbackURL)
	token, err := t.RequestToken(params)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(twitterAuthAPI, token.oauthToken), nil
}

// RequestToken requests token to twitter authentication.
func (t *TwitterClient) RequestToken(params url.Values) (*TwitterToken, error) {
	path := api + "/oauth/request_token"

	res, err := t.request(http.MethodPost, path, params, false)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)
	uv, err := url.ParseQuery(string(b))
	if err != nil {
		return nil, err
	}

	// verify callback
	verify := uv.Get("oauth_callback_confirmed")
	if verify != "true" {
		return nil, errors.New("callback is not verified")
	}

	return &TwitterToken{
		oauthToken:       uv.Get("oauth_token"),
		oauthTokenSecret: uv.Get("oauth_token_secret"),
	}, nil
}

// AccessToken requests access token.
func (t *TwitterClient) AccessToken(params url.Values) (*TwitterToken, error) {
	path := api + "/oauth/access_token"

	res, err := t.request(http.MethodPost, path, params, false)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)
	uv, err := url.ParseQuery(string(b))
	if err != nil {
		return nil, err
	}

	return &TwitterToken{
		oauthToken:       uv.Get("oauth_token"),
		oauthTokenSecret: uv.Get("oauth_token_secret"),
	}, nil
}

// VerifyUser verfies user, then set user information to argument interface.
func (t *TwitterClient) VerifyUser(user interface{}, params url.Values) error {
	return t.API(http.MethodGet, api+"/1.1/account/verify_credentials.json", params, user)
}

// Tweet tweets on timeline.
func (t *TwitterClient) Tweet(params url.Values) error {
	return t.API(http.MethodPost, api+"/1.1/statuses/update.json", params, nil)
}

// Home gets home timeline.
func (t *TwitterClient) Home(timeline interface{}, params url.Values) error {
	return t.API(http.MethodGet, api+"/1.1/statuses/home_timeline.json", params, timeline)
}

// API calls arbitary twitter api.
func (t *TwitterClient) API(method, path string, params url.Values, ret interface{}) error {
	res, err := t.request(method, path, params, true)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if ret != nil {
		if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
			return err
		}
	}
	return nil
}

// SetToken sets oauth token and oauth secret to twitter authentication.
func (t *TwitterClient) SetToken(token *TwitterToken) {
	t.oauthToken = token.oauthToken
	t.oauthTokenSecret = token.oauthTokenSecret
}

func (t *TwitterClient) request(method, path string, params url.Values, isAPI bool) (*http.Response, error) {
	var r io.Reader
	reqPath := path
	if params != nil {
		if isAPI {
			reqPath = path + "?" + params.Encode()
		} else {
			r = strings.NewReader(params.Encode())
		}
	}

	request, err := http.NewRequest(method, reqPath, r)
	if err != nil {
		return nil, err
	}
	if request.Body != nil {
		defer request.Body.Close()
	}

	header := t.getHeader(method, path, params)
	request.Header.Set("Authorization", header)
	res, err := t.client.Do(request)
	if err != nil {
		return nil, err
	} else if 400 <= res.StatusCode {
		b, _ := ioutil.ReadAll(res.Body)
		return nil, errors.New(string(b))
	}
	return res, nil
}

func (t *TwitterClient) getHeader(method, path string, params url.Values) string {
	header, _ := CreateTwitterOauthHeader(method, path, t.consumerKey, t.consumerSecret, t.oauthToken, t.oauthTokenSecret, params)
	return header
}

// CreateTwitterOauthHeader creates oauth header for twitter
func CreateTwitterOauthHeader(method, path, consumerKey, consumerSecret, accessToken, tokenSecret string, params url.Values) (string, string) {
	nonce := util.RandSeq(32)
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)

	elems := make([]string, 0, 6+len(params))
	elems = append(elems, "oauth_consumer_key="+consumerKey)
	elems = append(elems, "oauth_nonce="+nonce)
	elems = append(elems, "oauth_signature_method=HMAC-SHA1")
	elems = append(elems, "oauth_timestamp="+timestamp)
	elems = append(elems, "oauth_token="+accessToken)
	elems = append(elems, "oauth_version=1.0")
	for k := range params {
		v := params.Get(k)
		if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
			v = url.QueryEscape(v)
		} else {
			p := &url.URL{Path: v}
			v = p.String()
		}
		elems = append(elems, k+"="+v)
	}

	base := strings.ToUpper(method) + "&" + url.QueryEscape(path)
	sort.Strings(elems)
	q := url.QueryEscape(strings.Join(elems, "&"))

	baseString := base + "&" + q
	signingKey := consumerSecret + "&" + tokenSecret

	signature := generateSignature(baseString, signingKey)

	ckh := "oauth_consumer_key=\"" + consumerKey + "\""
	nh := "oauth_nonce=\"" + nonce + "\""
	sh := "oauth_signature=\"" + signature + "\""
	smh := "oauth_signature_method=\"HMAC-SHA1\""
	tsh := "oauth_timestamp=\"" + timestamp + "\""
	th := "oauth_token=\"" + accessToken + "\""
	vh := "oauth_version=\"1.0\""

	value := strings.Join([]string{ckh, nh, sh, smh, tsh, th, vh}, ", ")
	return "OAuth " + value, baseString
}

func generateSignature(baseString, signingKey string) string {
	p := []byte(baseString)
	hash := hmac.New(sha1.New, []byte(signingKey))
	hash.Write(p)
	sum := hash.Sum(nil)
	return url.QueryEscape(base64.StdEncoding.EncodeToString(sum))
}
