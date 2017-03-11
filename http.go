package gko

import (
	"strings"
)

type device int

const (
	iPhone device = iota
	iPod
	iPad
	androidPhone
	androidTablet
	browser
)

// IsMobile return true if http user agent is mobile.
func IsMobile(ua string) bool {
	d := getDevice(ua)
	if d == browser {
		return false
	}
	return true
}

// IsBrowser return true if http user agent is browser.
func IsBrowser(ua string) bool {
	d := getDevice(ua)
	if d == browser {
		return true
	}
	return false
}

func getDevice(ua string) device {
	if strings.Contains(ua, "Apple-iPhone") {
		return iPhone
	} else if strings.Contains(ua, "Apple-iPod") {
		return iPod
	} else if strings.Contains(ua, "Apple-iPad") {
		return iPad
	} else if strings.Contains(ua, "Android") && strings.Contains(ua, "Mobile") {
		return androidPhone
	} else if strings.Contains(ua, "Android") {
		return androidTablet
	}
	return browser
}
