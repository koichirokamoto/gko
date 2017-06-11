package push

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/koichirokamoto/gko/log"
	"github.com/koichirokamoto/webpush"
	"github.com/sideshow/apns2"
	"google.golang.org/api/gensupport"
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
			err := retry(w.work, gensupport.DefaultBackoffStrategy())
			if err != nil {
				log.DefaultLogger.Log(log.Error, err.Error())
			}
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
	c       *http.Client
	key     string
	sub     *FcmSubscription
	payload []byte
}

// SendFcmNotification push fcm notification to user.
func SendFcmNotification(c *http.Client, key string, subs []*FcmSubscription, payload []byte) {
	in := make(chan worker)
	go func() {
		for _, sub := range subs {
			in <- &fcmWorker{c, key, sub, payload}
		}
		close(in)
	}()

	runWorker(in)
}

func (f *fcmWorker) work() error {
	encryption, err := webpush.Encryption(f.sub.Key, f.sub.Auth, f.payload, 0)
	if err != nil {
		log.DefaultLogger.Log(log.Error, err.Error())
		return err
	}

	endpoint := strings.Replace(f.sub.Endpoint, gcmURL, fcmURL, 1)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(encryption.Payload))
	if err != nil {
		log.DefaultLogger.Log(log.Error, err.Error())
		return err
	}
	defer req.Body.Close()
	req.Header.Set("Authorization", "key="+f.key)
	req.Header.Set("Encryption", "salt="+base64.URLEncoding.EncodeToString(encryption.Salt))
	req.Header.Set("Crypto-Key", "dh="+base64.URLEncoding.EncodeToString(encryption.PublickKey))
	req.Header.Set("Content-Encoding", "aesgcm")
	req.ContentLength = int64(len(encryption.Payload))
	res, err := f.c.Do(req)
	if err != nil {
		log.DefaultLogger.Log(log.Error, err.Error())
		return err
	}
	res.Body.Close()
	return nil
}

type apnsWorker struct {
	c           *apns2.Client
	deviceToken string
	payload     []byte
}

// SendApns2Notification send apns2 notification.
func SendApns2Notification(cert tls.Certificate, deviceTokens []string, payload []byte) {
	in := make(chan worker)

	go func() {
		c := apns2.NewClient(cert)
		for _, d := range deviceTokens {
			in <- &apnsWorker{c, d, payload}
		}
		close(in)
	}()

	runWorker(in)
}

func (a *apnsWorker) work() error {
	notification := &apns2.Notification{}
	notification.DeviceToken = a.deviceToken
	notification.Payload = a.payload

	res, err := a.c.Push(notification)
	if err != nil {
		log.DefaultLogger.Log(log.Error, err.Error())
		return err
	}

	if !res.Sent() {
		errMsg := fmt.Sprintf("not sent: %d %s %s", res.StatusCode, res.ApnsID, res.Reason)
		log.DefaultLogger.Log(log.Error, errMsg)
		return fmt.Errorf(errMsg)
	}

	return nil
}
