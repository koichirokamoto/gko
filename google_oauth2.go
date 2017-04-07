package gko

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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

// GCPConfigKey is google cloud platform config key.
type GCPConfigKey int

// GCPProjectID is google cloud platform project id key.
var GCPProjectID GCPConfigKey = 1

func getDefaultTokenSource(ctx context.Context, scopes ...string) (oauth2.TokenSource, string, error) {
	projectID, ok := ctx.Value(GCPProjectID).(string)
	if !ok {
		return nil, "", errors.New("project id is not in context")
	}

	t, err := google.DefaultTokenSource(ctx, scopes...)
	if err != nil {
		return nil, "", err
	}

	return t, projectID, nil
}
