package mail

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sendgrid/rest"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	gmail "google.golang.org/api/gmail/v1"
	gaemail "google.golang.org/appengine/mail"
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

// SendGridClientFactory is sendgrid client factory interface.
type SendGridClientFactory interface {
	New(*http.Client, string) Mail
}

// Mail is mail interface.
type Mail interface {
	Send(string, string, string, string, []string) error
}

// sendGridMailFactoryImpl implements mail factory interface.
type sendGridMailFactoryImpl struct{}

// New return new send grid mail.
func (s *sendGridMailFactoryImpl) New(client *http.Client, key string) Mail {
	return newSendGridMail(client, key)
}

// sendGridMailClient is mail client of sendgrid interface.
type sendGridMailClient struct {
	client *http.Client
	key    string
}

func newSendGridMail(client *http.Client, key string) Mail {
	return &sendGridMailClient{client, key}
}

// Send send email using sendgrid.
func (s *sendGridMailClient) Send(from, subject, content, contentType string, to []string) error {
	req := sendgrid.GetRequest(s.key, endpoint, host)
	req.Method = http.MethodPost
	req.Body = mail.GetRequestBody(s.buildSendGridMail(from, subject, content, contentType, to))

	httpreq, err := rest.BuildRequestObject(req)
	if err != nil {
		return err
	}
	res, err := s.client.Do(httpreq)
	if err != nil {
		return err
	} else if 400 <= res.StatusCode {
		msg, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("status code is in error range, %s", msg)
	}
	return nil
}

func (s *sendGridMailClient) buildSendGridMail(from, subject, content, contentType string, to []string) *mail.SGMailV3 {
	sg := mail.NewV3Mail()
	sg.SetFrom(mail.NewEmail("", from))
	sg.Subject = subject
	sg.AddContent(mail.NewContent(contentType, content))
	p := mail.NewPersonalization()
	for _, t := range to {
		mailto := mail.NewEmail("", t)
		p.AddTos(mailto)
	}
	sg.AddPersonalizations(p)
	return sg
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

// GmailClientFactory is gmail client factory interface.
type GmailClientFactory interface {
	New(context.Context, *oauth2.Config, string) Mail
}

// GetGmailFactory return gmail factory.
func GetGmailFactory() GmailClientFactory {
	if gmailFactory == nil {
		gmailFactory = &gmailFactoryImpl{}
	}
	return gmailFactory
}

type gmailFactoryImpl struct{}

func (g *gmailFactoryImpl) New(ctx context.Context, conf *oauth2.Config, refreshToken string) Mail {
	srv, err := NewGmailService(ctx, conf, refreshToken)
	if err != nil {
		return nil
	}
	return &gmailClient{srv}
}

type gmailClient struct {
	srv *gmail.Service
}

func (g *gmailClient) Send(from, subject, content, contentType string, to []string) error {
	var headers []*gmail.MessagePartHeader
	part := &gmail.MessagePartBody{}
	body := &gmail.MessagePart{
		Headers:  headers,
		Body:     part,
		MimeType: contentType,
	}
	msg := &gmail.Message{
		Id:      RandSeq(32),
		Payload: body,
		Raw:     "",
	}
	g.srv.Users.Messages.Send(from, msg).Do()
	return nil
}
