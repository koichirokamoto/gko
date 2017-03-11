package gko

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

// FirebaseAuthUserID return firebase auth user id.
func FirebaseAuthUserID(r *http.Request, id string) (uid string, err error) {
	keyLists := "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"
	url := "https://securetoken.google.com/"
	token, err := request.ParseFromRequestWithClaims(r, request.ArgumentExtractor{"token"}, jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
		res, err := urlfetch.Client(appengine.NewContext(r)).Get(keyLists)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		var keyList map[string]string
		if err := json.NewDecoder(res.Body).Decode(keyList); err != nil {
			return nil, err
		}

		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid is not string")
		}

		keyString := keyList[kid]
		block, _ := pem.Decode([]byte(keyString))
		cert, _ := x509.ParseCertificate(block.Bytes)
		pubKey := cert.PublicKey
		return pubKey, nil
	})
	if err != nil {
		return
	}

	claims, ok := token.Claims.(jwt.StandardClaims)
	if !ok {
		err = fmt.Errorf("token[%#v] is something wrong", token)
		return
	}

	if claims.Audience != id {
		err = errors.New("firebase project id is not matched")
		return
	} else if claims.Issuer != fmt.Sprintf(url+"%s", id) {
		err = errors.New("firebase project id is not matched")
		return
	}

	uid = claims.Subject
	return
}
