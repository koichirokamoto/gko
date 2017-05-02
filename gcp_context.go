package gko

import (
	"errors"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// gcpConfigKey is google cloud platform config key.
type gcpConfigKey int

// gcpProjectID is google cloud platform project id key.
var gcpProjectID gcpConfigKey = 1

// GCPDefaultContext create gcp default context from context.
func GCPDefaultContext(parent context.Context, projectID string) context.Context {
	return context.WithValue(parent, gcpProjectID, projectID)
}

func getDefaultTokenSource(ctx context.Context, scopes ...string) (oauth2.TokenSource, string, error) {
	projectID, ok := ctx.Value(gcpProjectID).(string)
	if !ok {
		return nil, "", errors.New("project id is not in context")
	}

	t, err := google.DefaultTokenSource(ctx, scopes...)
	if err != nil {
		return nil, "", err
	}

	return t, projectID, nil
}
