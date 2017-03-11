package gko

import (
	"golang.org/x/oauth2"
)

const (
	authURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL = "https://www.googleapis.com/oauth2/v4/token"
)

// GoogleOAuth2Config return google oauth2 config.
func GoogleOAuth2Config(clientID, clientSecret, redirectURL string) oauth2.Config {
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
