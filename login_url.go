package gko

import (
	"fmt"
	"net/url"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const (
	twitterAuthAPI  = "https://api.twitter.com/oauth/authenticate?oauth_token=%s"
	facebookAuthAPI = "https://www.facebook.com/dialog/oauth?"
)

// GoogleLoginURL return google login url.
func GoogleLoginURL(state, clientID, clientSecret, redirectURL string, opts ...oauth2.AuthCodeOption) string {
	conf := GoogleOAuth2Config(clientID, clientSecret, redirectURL)
	return conf.AuthCodeURL(state, opts...)
}

// TwitterLoginURL return twitter login url.
func TwitterLoginURL(c context.Context, consumerKey, consumerSecret, oauthToken, oauthTokenSecret, callback string) (string, error) {
	twitter := NewTwitterAPI(c, consumerKey, consumerSecret, oauthToken, oauthTokenSecret)

	params := url.Values{}
	params.Set("oauth_callback", callback)
	token, err := twitter.RequestToken(params)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(twitterAuthAPI, token.oauthToken), nil
}

// FacebookLoginURL return facebook login url.
func FacebookLoginURL(appID, callback string) string {
	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("redirect_uri", callback)
	params.Set("auth_type", "rerequest")
	params.Set("scope", "public_profile")
	params.Set("state", RandSeq(32))
	return facebookAuthAPI + params.Encode()
}
