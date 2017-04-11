package gko

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
	_ MailFactory = (*sendGridMailFactoryImpl)(nil)
	_ MailFactory = (*gaeMailFactoryImpl)(nil)

	_ Mail = (*sendGridMailClient)(nil)
	_ Mail = (*gaeMailClient)(nil)
	_ Mail = (*gmailClient)(nil)
)

var mailFactory MailFactory

// GetSendGridMailFactory return sendgrid mail factory.
func GetSendGridMailFactory(client *http.Client) MailFactory {
	if mailFactory == nil {
		mailFactory = &sendGridMailFactoryImpl{client}
	}
	_, ok := mailFactory.(*sendGridMailFactoryImpl)
	if !ok {
		mailFactory = &sendGridMailFactoryImpl{client}
	}
	return mailFactory
}

// GetGAEMailFactory return gae mail factory.
func GetGAEMailFactory() MailFactory {
	if mailFactory == nil {
		mailFactory = &gaeMailFactoryImpl{}
	}
	_, ok := mailFactory.(*gaeMailFactoryImpl)
	if !ok {
		mailFactory = &gaeMailFactoryImpl{}
	}
	return mailFactory
}

// GetGmailFactory return gmail factory.
func GetGmailFactory(conf *oauth2.Config) MailFactory {
	if mailFactory == nil {
		mailFactory = &gmailFactoryImpl{conf}
	}
	_, ok := mailFactory.(*gmailFactoryImpl)
	if !ok {
		mailFactory = &gmailFactoryImpl{conf}
	}
	return mailFactory
}

// MailFactory is mail factory interface.
type MailFactory interface {
	New(context.Context, string) Mail
}

// Mail is mail interface.
type Mail interface {
	Send(string, string, string, string, []string) error
}

// sendGridMailFactoryImpl implements mail factory interface.
type sendGridMailFactoryImpl struct {
	client *http.Client
}

// New return new send grid mail.
func (s *sendGridMailFactoryImpl) New(ctx context.Context, key string) Mail {
	return newSendGridMail(ctx, s.client, key)
}

// sendGridMailClient is mail client of sendgrid interface.
type sendGridMailClient struct {
	ctx    context.Context
	client *http.Client
	key    string
}

func newSendGridMail(ctx context.Context, client *http.Client, key string) Mail {
	return &sendGridMailClient{ctx, client, key}
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

type gaeMailFactoryImpl struct{}

func (g *gaeMailFactoryImpl) New(ctx context.Context, appID string) Mail {
	return newGAEMailClient(ctx, appID)
}

type gaeMailClient struct {
	ctx   context.Context
	appID string
}

func newGAEMailClient(ctx context.Context, appID string) *gaeMailClient {
	return &gaeMailClient{ctx, appID}
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

type gmailFactoryImpl struct {
	conf *oauth2.Config
}

func (g *gmailFactoryImpl) New(ctx context.Context, refreshToken string) Mail {
	srv, err := NewGmailService(ctx, g.conf, refreshToken)
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
