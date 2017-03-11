package gko

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// SafeURL return url safe strings.
func SafeURL(s string) string {
	s = strings.Replace(s, "+", "-", -1)
	s = strings.Replace(s, "/", "_", -1)
	s = strings.Replace(s, "=", "", -1)
	return s
}

// Retry is interface do retry.
type Retry interface {
	DoRetry() error
	HandleError(error) bool
}

// ExponentialBackoff is implementation of exponential backoff.
func ExponentialBackoff(r Retry, count uint) error {
	var tried uint
	if count <= 0 {
		count = 1
	}

	var err error
	for tried < count {
		if err = r.DoRetry(); err != nil && r.HandleError(err) {
			WaitExponentialTime(tried)
			tried++
			continue
		}
		break
	}
	return err
}

// WaitExponentialTime stop thread during exponential time based on argument.
func WaitExponentialTime(count uint) {
	rand.Seed(time.Now().UnixNano())
	t := time.After(time.Duration((1<<count)*1000+rand.Intn(10)*100) * time.Millisecond)
	select {
	case <-t:
		return
	}
}

// UUID generate string formatted by uuid.
func UUID() string {
	return uuid.NewV4().String()
}

// ProcessRecursively process directory using f recursively.
func ProcessRecursively(dir string, f func(d string) error) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("failt to open directory: %s", err)
	}
	i, err := d.Stat()
	if err != nil {
		return fmt.Errorf("fail to get file stat: %s", err)
	}
	if !i.IsDir() {
		return fmt.Errorf("%s is not directory", dir)
	}

	if err := f(dir); err != nil {
		return err
	}

	ds, err := d.Readdir(0)
	if err != nil {
		return err
	}
	for _, d := range ds {
		if d.IsDir() {
			if err := ProcessRecursively(d.Name(), f); err != nil {
				return err
			}
		}
	}
	return nil
}

// SliceRetrivedDuplicate returns new slice retrived duplicate data from slice.
func SliceRetrivedDuplicate(slice []string) []string {
	m := make(map[string]bool)
	newSlice := make([]string, 0, len(slice))
	for _, s := range slice {
		if m[s] == false {
			m[s] = true
			newSlice = append(newSlice, s)
		}
	}

	ns := newSlice
	return ns[:len(ns)]
}

// SliceToBoolMap changes slice to map.
func SliceToBoolMap(slice []string) map[string]bool {
	m := make(map[string]bool)
	for _, s := range slice {
		m[s] = true
	}
	return m
}

// UTF8ToSJIS is convert utf8 to shiftjis.
func UTF8ToSJIS(w io.Writer) io.Writer {
	return transform.NewWriter(w, japanese.ShiftJIS.NewEncoder())
}

// SJISToUTF8 is convert shiftjis to utf8.
func SJISToUTF8(r io.Reader) io.Reader {
	return transform.NewReader(r, japanese.ShiftJIS.NewDecoder())
}
