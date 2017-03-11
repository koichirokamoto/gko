package gko

import (
	"net/http"

	admin "google.golang.org/api/admin/directory/v1"
	calendar "google.golang.org/api/calendar/v3"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func newClientFromRefreshToken(ctx context.Context, conf oauth2.Config, refreshToken string) *http.Client {
	token := &oauth2.Token{RefreshToken: refreshToken}
	return conf.Client(ctx, token)
}

// NewCalendarService return new calendar service.
func NewCalendarService(ctx context.Context, conf oauth2.Config, refreshToken string) (*calendar.Service, error) {
	return calendar.New(newClientFromRefreshToken(ctx, conf, refreshToken))
}

// NewAdminDirectoryService return new admin directory service.
func NewAdminDirectoryService(ctx context.Context, conf oauth2.Config, refreshToken string) (*admin.Service, error) {
	return admin.New(newClientFromRefreshToken(ctx, conf, refreshToken))
}
