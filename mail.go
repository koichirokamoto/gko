package gko

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sendgrid/rest"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/net/context"
	"google.golang.org/appengine/urlfetch"
)

var (
	endpoint = "/v3/mail/send"
	host     = "https://api.sendgrid.com"
)

// Factory is mail factory interface.
type Factory interface {
	New(context.Context) Mail
}

// Mail is mail interface.
type Mail interface {
	Send(string, string, string, string, []string) error
}

// SendGridMailFactory implements mail factory interface.
type SendGridMailFactory struct{}

// New return new send grid mail.
func (s *SendGridMailFactory) New(ctx context.Context, key string) Mail {
	return newSendGridMail(ctx, key)
}

// SendGridMailClient is mail client of sendgrid interface.
type SendGridMailClient struct {
	ctx context.Context
	key string
}

func newSendGridMail(ctx context.Context, key string) Mail {
	return &SendGridMailClient{ctx, key}
}

// Send send email using sendgrid.
func (s *SendGridMailClient) Send(from, subject, content, contentType string, to []string) error {
	req := sendgrid.GetRequest(s.key, endpoint, host)
	req.Method = http.MethodPost
	req.Body = mail.GetRequestBody(s.buildSendGridMail(from, subject, content, contentType, to))

	httpreq, err := rest.BuildRequestObject(req)
	if err != nil {
		ErrorLog(s.ctx, err.Error())
		return err
	}
	res, err := urlfetch.Client(s.ctx).Do(httpreq)
	if err != nil {
		ErrorLog(s.ctx, err.Error())
		return err
	} else if 400 <= res.StatusCode {
		msg, err := ioutil.ReadAll(res.Body)
		if err != nil {
			ErrorLog(s.ctx, err.Error())
			return err
		}
		ErrorLog(s.ctx, "response status is in error range, %d, %s", res.StatusCode, msg)
		return fmt.Errorf("status code is in error range")
	}
	return nil
}

func (s *SendGridMailClient) buildSendGridMail(from, subject, content, contentType string, to []string) *mail.SGMailV3 {
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
