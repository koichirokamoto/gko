package gko

import (
	"bytes"

	"crypto/tls"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/RobotsAndPencils/buford/certificate"
	"github.com/koichirokamoto/apns"
	"github.com/koichirokamoto/webpush"
	"golang.org/x/net/context"
	"google.golang.org/appengine/socket"
	"google.golang.org/appengine/urlfetch"
)

const (
	fcmURL = "https://fcm.googleapis.com/fcm/send"
	gcmURL = "https://android.googleapis.com/gcm/send"
)

type worker interface {
	work() error
	handleErr() <-chan struct{}
}

func runWorker(wrk <-chan worker) {
	retried := make([](<-chan struct{}), 0)
	for w := range wrk {
		if err := w.work(); err != nil {
			retried = append(retried, w.handleErr())
		}
	}

	for _, r := range retried {
		<-r
	}
}

// FcmSub is fcm subscription.
type FcmSub struct {
	Endpoint string
	Key      string
	Auth     string
}

// SendFcmNotification push fcm notification to user.
func SendFcmNotification(ctx context.Context, subs []*FcmSub, payload []byte) error {
	serverKey := ""
	in := make(chan worker)
	go func() {
		for _, sub := range subs {
			in <- &fcmWorker{ctx, sub, payload, serverKey, 0, 5}
		}
		close(in)
	}()

	InfoLog(ctx, "Send fcm notification worker start...")
	runWorker(in)
	InfoLog(ctx, "Send fcm notification worker end...")
	return nil
}

type fcmWorker struct {
	ctx       context.Context
	sub       *FcmSub
	payload   []byte
	serverKey string
	count     uint
	limit     uint
}

func (f *fcmWorker) work() error {
	encryption, err := webpush.Encryption(f.sub.Key, f.sub.Auth, f.payload, 0)
	if err != nil {
		return err
	}

	endpoint := strings.Replace(f.sub.Endpoint, gcmURL, fcmURL, 1)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(encryption.Payload))
	if err != nil {
		return err
	}
	defer req.Body.Close()
	req.Header.Set("Authorization", "key="+f.serverKey)
	req.Header.Set("Encryption", "salt="+base64.URLEncoding.EncodeToString(encryption.Salt))
	req.Header.Set("Crypto-Key", "dh="+base64.URLEncoding.EncodeToString(encryption.PublickKey))
	req.Header.Set("Content-Encoding", "aesgcm")
	req.ContentLength = int64(len(encryption.Payload))
	res, err := urlfetch.Client(f.ctx).Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

func (f *fcmWorker) handleErr() <-chan struct{} {
	fin := make(chan struct{}, 1)
	go func() {
		for f.count < f.limit {
			if err := f.work(); err != nil {
				WaitExponentialTime(f.count)
				continue
			}
		}
		close(fin)
	}()
	return fin
}

const apnsGateway = "gateway.push.apple.com:2195"

type apnsConfig struct {
	filename string
	password string
	host     string
}

// SendApnsNotification send apns2 notification.
func SendApnsNotification(ctx context.Context, deviceTokens []string, config *apnsConfig, notification *apns.PushNotification) error {
	in := make(chan worker)
	go func() {
		for _, d := range deviceTokens {
			payload, err := notification.ToBytes(d)
			if err != nil {
				WarningLog(ctx, err.Error())
				continue
			}
			in <- &apnsWorker{ctx, config, payload, 0, 5}
		}
		close(in)
	}()

	InfoLog(ctx, "Send apns notification worker start...")
	runWorker(in)
	InfoLog(ctx, "Send apns notification worker end...")
	return nil
}

type apnsWorker struct {
	ctx     context.Context
	config  *apnsConfig
	payload []byte
	count   uint
	limit   uint
}

func (a *apnsWorker) work() error {
	cert, err := certificate.Load(a.config.filename, a.config.password)
	if err != nil {
		ErrorLog(a.ctx, err.Error())
		return err
	}

	tlsconfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   strings.Split(apnsGateway, ":")[0],
	}

	conn, err := socket.Dial(a.ctx, "tcp", apnsGateway)
	if err != nil {
		ErrorLog(a.ctx, err.Error())
		return err
	}

	tlsconn := tls.Client(conn, tlsconfig)
	if err := tlsconn.Handshake(); err != nil {
		ErrorLog(a.ctx, err.Error())
		return err
	}
	defer tlsconn.Close()

	if _, err := tlsconn.Write(a.payload); err != nil {
		ErrorLog(a.ctx, err.Error())
		return err
	}

	// TODO: handle apns feedback or error
	return nil
}

func (a *apnsWorker) handleErr() <-chan struct{} {
	fin := make(chan struct{}, 1)
	go func() {
		for a.count < a.limit {
			if err := a.work(); err != nil {
				WaitExponentialTime(a.count)
				a.count++
				continue
			}
		}
		close(fin)
	}()
	return fin
}
