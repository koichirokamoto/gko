package mail

import (
	"golang.org/x/net/context"
	gaemail "google.golang.org/appengine/mail"
)

type gaeMailClient struct {
	ctx context.Context
}

func newGAEMailClient(ctx context.Context) *gaeMailClient {
	return &gaeMailClient{ctx}
}

func (g *gaeMailClient) Send(from, subject, content, contentType string, to []string) error {
	msg := &gaemail.Message{
		Sender:  from,
		To:      to,
		Subject: subject,
	}
	if contentType == "text/html" {
		msg.HTMLBody = content
	} else {
		msg.Body = content
	}
	return gaemail.Send(g.ctx, msg)
}
