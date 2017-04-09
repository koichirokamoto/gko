package gko

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sendgrid/rest"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	gaemail "google.golang.org/appengine/mail"
	"google.golang.org/appengine/urlfetch"
)

var (
	endpoint = "/v3/mail/send"
	host     = "https://api.sendgrid.com"
)

var (
	_ MailFactory = (*sendGridMailFactoryImpl)(nil)
	_ MailFactory = (*gaeMailFactoryImpl)(nil)
)

var (
	sendgridMailFactory MailFactory
	gaeMailFactory      MailFactory
)

// GetSendGridMailFactory return sendgrid mail factory.
func GetSendGridMailFactory() MailFactory {
	if sendgridMailFactory == nil {
		sendgridMailFactory = &sendGridMailFactoryImpl{}
	}
	return sendgridMailFactory
}

// GetGAEMailFactory return gae mail factory.
func GetGAEMailFactory() MailFactory {
	if gaeMailFactory == nil {

	}
	return gaeMailFactory
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
type sendGridMailFactoryImpl struct{}

// New return new send grid mail.
func (s *sendGridMailFactoryImpl) New(ctx context.Context, key string) Mail {
	return newSendGridMail(ctx, key)
}

// sendGridMailClient is mail client of sendgrid interface.
type sendGridMailClient struct {
	ctx context.Context
	key string
}

func newSendGridMail(ctx context.Context, key string) Mail {
	return &sendGridMailClient{ctx, key}
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
	res, err := urlfetch.Client(s.ctx).Do(httpreq)
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
	if appID == "" {
		appID = appengine.AppID(ctx)
	}
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
