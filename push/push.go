package push

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/koichirokamoto/webpush"
	"github.com/sideshow/apns2"
	"golang.org/x/net/context"
	"google.golang.org/api/gensupport"
)

const (
	fcmURL = "https://fcm.googleapis.com/fcm/send"
	gcmURL = "https://android.googleapis.com/gcm/send"
)

var (
	// ServerKey is fcm server key.
	ServerKey string
	// Cert is apns notification certificate.
	Cert tls.Certificate
	// Backoff is backoff strategy.
	Backoff gensupport.BackoffStrategy
	// DevEnv is flag whether environment is development.
	DevEnv bool
)

type worker interface {
	work() error
}

func runWorker(wrk <-chan worker) {
	if Backoff == nil {
		Backoff = gensupport.DefaultBackoffStrategy()
	}
	var wg sync.WaitGroup
	for w := range wrk {
		wg.Add(1)
		go func(w worker) {
			defer wg.Done()
			retry(w.work, Backoff)
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

type fcmWorker struct {
	ctx     context.Context
	sub     *FcmSubscription
	payload []byte
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
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	return nil
}

type apnsWorker struct {
	ctx         context.Context
	deviceToken string
	payload     []byte
}

// SendApnsNotification send apns2 notification.
func SendApnsNotification(ctx context.Context, deviceTokens []string, payload []byte) {
	in := make(chan worker)

	go func() {
		for _, d := range deviceTokens {
			in <- &apnsWorker{ctx, d, payload}
		}
		close(in)
	}()

	runWorker(in)
}

func (a *apnsWorker) work() error {
	var client *apns2.Client
	if DevEnv {
		client = apns2.NewClient(Cert).Development()
	} else {
		client = apns2.NewClient(Cert).Production()
	}

	notification := &apns2.Notification{}
	notification.DeviceToken = a.deviceToken
	notification.Payload = a.payload

	res, err := client.Push(notification)
	if err != nil {
		return err
	}

	if !res.Sent() {
		return fmt.Errorf("not sent: %v %v %v", res.StatusCode, res.ApnsID, res.Reason)
	}

	return nil
}
