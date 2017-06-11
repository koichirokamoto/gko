package mail

import (
	"net/http"

	"github.com/koichirokamoto/gko/cloud/gsuite"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var (
	endpoint = "/v3/mail/send"
	host     = "https://api.sendgrid.com"
)

var (
	_ SendGridClientFactory = (*sendGridMailFactoryImpl)(nil)
	_ GAEMailClientFactory  = (*gaeMailFactoryImpl)(nil)
	_ GmailClientFactory    = (*gmailFactoryImpl)(nil)

	_ Mail = (*sendGridMailClient)(nil)
	_ Mail = (*gaeMailClient)(nil)
	_ Mail = (*gmailClient)(nil)
)

var (
	sendgridmailFactory SendGridClientFactory
	gaemailFactory      GAEMailClientFactory
	gmailFactory        GmailClientFactory
)

// GetSendGridMailFactory return sendgrid mail factory.
func GetSendGridMailFactory() SendGridClientFactory {
	if sendgridmailFactory == nil {
		sendgridmailFactory = &sendGridMailFactoryImpl{}
	}
	return sendgridmailFactory
}

// Mail is mail interface.
type Mail interface {
	Send(string, string, string, string, []string) error
}

// SendGridClientFactory is sendgrid client factory interface.
type SendGridClientFactory interface {
	New(*http.Client, string) Mail
}

// sendGridMailFactoryImpl implements mail factory interface.
type sendGridMailFactoryImpl struct{}

// New return new send grid mail.
func (s *sendGridMailFactoryImpl) New(client *http.Client, key string) Mail {
	return newSendGridMail(client, key)
}

// GAEMailClientFactory is gae mail client factory interface.
type GAEMailClientFactory interface {
	New(context.Context) Mail
}

// GetGAEMailFactory return gae mail factory.
func GetGAEMailFactory() GAEMailClientFactory {
	if gaemailFactory == nil {
		gaemailFactory = &gaeMailFactoryImpl{}
	}
	return gaemailFactory
}

type gaeMailFactoryImpl struct{}

func (g *gaeMailFactoryImpl) New(ctx context.Context) Mail {
	return newGAEMailClient(ctx)
}

// GmailClientFactory is gmail client factory interface.
type GmailClientFactory interface {
	New(context.Context, *oauth2.Config, string) (Mail, error)
}

// GetGmailFactory return gmail factory.
func GetGmailFactory() GmailClientFactory {
	if gmailFactory == nil {
		gmailFactory = &gmailFactoryImpl{}
	}
	return gmailFactory
}

type gmailFactoryImpl struct{}

func (g *gmailFactoryImpl) New(ctx context.Context, conf *oauth2.Config, refreshToken string) (Mail, error) {
	srv, err := gsuite.NewGmailService(ctx, conf, refreshToken)
	if err != nil {
		return nil, err
	}
	return &gmailClient{srv}, nil
}
