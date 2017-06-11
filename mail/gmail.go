package mail

import (
	"github.com/koichirokamoto/gko/util"
	gmail "google.golang.org/api/gmail/v1"
)

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
		Id:      util.RandSeq(32),
		Payload: body,
		Raw:     "",
	}
	_, err := g.srv.Users.Messages.Send(from, msg).Do()
	return err
}
