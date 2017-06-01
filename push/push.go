package push

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"

	"github.com/koichirokamoto/apns"
	"github.com/koichirokamoto/webpush"
	"golang.org/x/net/context"
	"google.golang.org/api/gensupport"
	"google.golang.org/appengine/socket"
	"google.golang.org/appengine/urlfetch"
)

const (
	fcmURL = "https://fcm.googleapis.com/fcm/send"
	gcmURL = "https://android.googleapis.com/gcm/send"
)

var (
	// ServerKey is fcm server key.
	ServerKey string
	// ApnsCert is apns notification certificate.
	ApnsCert tls.Certificate
)

type worker interface {
	work() error
}

func runWorker(wrk <-chan worker) {
	var wg sync.WaitGroup
	for w := range wrk {
		wg.Add(1)
		go func(w worker) {
			defer wg.Done()
			retry(w.work, gensupport.DefaultBackoffStrategy())
		}(w)
	}
	wg.Wait()
}

// FcmSubscription is fcm subscription.
type FcmSubscription struct {
	Endpoint string
	Key      string
	Auth     string
}

// SendFcmNotification push fcm notification to user.
func SendFcmNotification(ctx context.Context, subs []*FcmSubscription, payload []byte) {
	in := make(chan worker)
	go func() {
		for _, sub := range subs {
			in <- &fcmWorker{ctx, sub, payload}
		}
		close(in)
	}()

	runWorker(in)
}

type fcmWorker struct {
	ctx     context.Context
	sub     *FcmSubscription
	payload []byte
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
	req.Header.Set("Authorization", "key="+ServerKey)
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

const apnsGateway = "gateway.push.apple.com:2195"

// SendApnsNotification send apns2 notification.
func SendApnsNotification(ctx context.Context, deviceTokens []string, notification *apns.PushNotification) {
	in := make(chan worker)

	tlsconfig := &tls.Config{
		Certificates: []tls.Certificate{ApnsCert},
		ServerName:   strings.Split(apnsGateway, ":")[0],
	}
	go func() {
		for _, d := range deviceTokens {
			payload, err := notification.ToBytes(d)
			if err != nil {
				continue
			}
			in <- &apnsWorker{ctx, tlsconfig, payload}
		}
		close(in)
	}()

	runWorker(in)
}

type apnsWorker struct {
	ctx       context.Context
	tlsconfig *tls.Config
	payload   []byte
}

func (a *apnsWorker) work() error {
	conn, err := socket.Dial(a.ctx, "tcp", apnsGateway)
	if err != nil {
		return err
	}

	tlsconn := tls.Client(conn, a.tlsconfig)
	if err := tlsconn.Handshake(); err != nil {
		return err
	}
	defer tlsconn.Close()

	if _, err := tlsconn.Write(a.payload); err != nil {
		return err
	}

	// TODO: handle apns feedback or error
	return nil
}
