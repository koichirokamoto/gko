package mail

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/koichirokamoto/gko/log"
	"github.com/sendgrid/rest"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/net/context"
)

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
		log.DefaultLogger.Log(s.ctx, log.Error, err.Error())
		return err
	}
	res, err := s.client.Do(httpreq)
	if err != nil {
		log.DefaultLogger.Log(s.ctx, log.Error, err.Error())
		return err
	} else if 400 <= res.StatusCode {
		msg, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.DefaultLogger.Log(s.ctx, log.Error, err.Error())
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
