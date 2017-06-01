package util

import (
	"strings"
)

type Device int

const (
	IPhone Device = iota
	IPod
	IPad
	AndroidPhone
	AndroidTablet
	Browser
)

// IsMobile return true if http user agent is mobile.
func IsMobile(ua string) bool {
	d := getDevice(ua)
	if d == Browser {
		return false
	}
	return true
}

// IsBrowser return true if http user agent is browser.
func IsBrowser(ua string) bool {
	d := getDevice(ua)
	if d == Browser {
		return true
	}
	return false
}

func GetDevice(ua string) Device {
	if strings.Contains(ua, "Apple-iPhone") {
		return IPhone
	} else if strings.Contains(ua, "Apple-iPod") {
		return IPod
	} else if strings.Contains(ua, "Apple-iPad") {
		return IPad
	} else if strings.Contains(ua, "Android") && strings.Contains(ua, "Mobile") {
		return AndroidPhone
	} else if strings.Contains(ua, "Android") {
		return AndroidTablet
	}
	return Browser
}
