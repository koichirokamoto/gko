package gko

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"

	"github.com/RobotsAndPencils/buford/certificate"
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

type worker interface {
	work() error
}

func runWorker(wrk <-chan worker) {
	var wg sync.WaitGroup
	for w := range wrk {
		wg.Add(1)
		go func(w worker) {
			defer wg.Done()
			Retry(w.work, gensupport.DefaultBackoffStrategy())
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
func SendFcmNotification(ctx context.Context, serverKey string, subs []*FcmSubscription, payload []byte) {
	in := make(chan worker)
	go func() {
		for _, sub := range subs {
			in <- &fcmWorker{ctx, sub, payload, serverKey}
		}
		close(in)
	}()

	runWorker(in)
}

type fcmWorker struct {
	ctx       context.Context
	sub       *FcmSubscription
	payload   []byte
	serverKey string
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

const apnsGateway = "gateway.push.apple.com:2195"

// ApnsConfig is apns notification setting.
type ApnsConfig struct {
	Filename string
	Password string
	Host     string
}

// SendApnsNotification send apns2 notification.
func SendApnsNotification(ctx context.Context, deviceTokens []string, config *ApnsConfig, notification *apns.PushNotification) {
	in := make(chan worker)
	go func() {
		for _, d := range deviceTokens {
			payload, err := notification.ToBytes(d)
			if err != nil {
				continue
			}
			in <- &apnsWorker{ctx, config, payload}
		}
		close(in)
	}()

	runWorker(in)
}

type apnsWorker struct {
	ctx     context.Context
	config  *ApnsConfig
	payload []byte
}

func (a *apnsWorker) work() error {
	cert, err := certificate.Load(a.config.Filename, a.config.Password)
	if err != nil {
		return err
	}

	tlsconfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   strings.Split(apnsGateway, ":")[0],
	}

	conn, err := socket.Dial(a.ctx, "tcp", apnsGateway)
	if err != nil {
		return err
	}

	tlsconn := tls.Client(conn, tlsconfig)
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
