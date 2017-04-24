package gko

import (
	"golang.org/x/oauth2"
)

const (
	authURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL = "https://www.googleapis.com/oauth2/v4/token"
)

// NewGoogleOAuth2Config return google oauth2 config.
func NewGoogleOAuth2Config(clientID, clientSecret, redirectURL string) oauth2.Config {
	oauth2Config := oauth2.Config{
		Scopes: []string{"openid", "email", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	oauth2Config.ClientID = clientID
	oauth2Config.ClientSecret = clientSecret
	oauth2Config.RedirectURL = redirectURL
	return oauth2Config
}

// GoogleLoginURL return google login url.
func GoogleLoginURL(state, clientID, clientSecret, redirectURL string, opts ...oauth2.AuthCodeOption) string {
	conf := NewGoogleOAuth2Config(clientID, clientSecret, redirectURL)
	return conf.AuthCodeURL(state, opts...)
}
