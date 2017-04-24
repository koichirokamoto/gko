package gko

import (
	"net/url"
)

const facebookAuthAPI = "https://www.facebook.com/dialog/oauth?"

// FacebookLoginURL return facebook login url.
func FacebookLoginURL(appID, callback string) string {
	params := url.Values{}
	params.Set("client_id", appID)
	params.Set("redirect_uri", callback)
	params.Set("auth_type", "rerequest")
	params.Set("scope", "public_profile")
	params.Set("state", RandSeq(32))
	return facebookAuthAPI + params.Encode()
}
