package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/securecookie"

	"github.com/couchbaselabs/consolio/types"
)

const (
	BROWSERID_ENDPOINT = "https://verifier.login.persona.org/verify"
	AUTH_COOKIE        = "consolio"
)

var NotAUser = errors.New("not a user")

var alphabet []byte

func init() {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789"

	for _, r := range letters {
		alphabet = append(alphabet, byte(r))
	}
}

func randstring(l int) string {
	stuff := make([]byte, l)
	_, err := io.ReadFull(rand.Reader, stuff)
	if err != nil {
		panic(err)
	}

	for i := range stuff {
		stuff[i] = alphabet[int(stuff[i])%len(alphabet)]
	}
	return string(stuff)
}

func getUser(email string) (consolio.User, error) {
	rv := consolio.User{}
	k := "u-" + email
	err := db.Get(k, &rv)
	if err == nil && rv.Type != "user" {
		return consolio.User{}, NotAUser
	}
	return rv, err
}

type browserIdData struct {
	Status   string
	Reason   string
	Email    string
	Audience string
	Expires  uint64
	Issuer   string
}

var secureCookie *securecookie.SecureCookie

func initSecureCookie(hashKey []byte) {
	secureCookie = securecookie.New(hashKey, nil)
}

func userFromCookie(cookie string) (consolio.User, error) {
	val := browserIdData{}
	err := secureCookie.Decode("consolio.User", cookie, &val)
	if err == nil {
		u, err := getUser(val.Email)
		if err != nil {
			u.Id = val.Email
		}
		return u, nil
	}
	return consolio.User{}, err
}

func whoami(r *http.Request) consolio.User {
	if cookie, err := r.Cookie(AUTH_COOKIE); err == nil {
		u, err := userFromCookie(cookie.Value)
		if err == nil {
			return u
		}
	}
	if ahdr := r.Header.Get("Authorization"); ahdr != "" {
		parts := strings.Split(ahdr, " ")
		if len(parts) < 2 {
			return consolio.User{}
		}
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return consolio.User{}
		}
		userpass := strings.SplitN(string(decoded), ":", 2)

		user, err := getUser(userpass[0])
		if err != nil {
			return consolio.User{}
		}

		if user.AuthToken == userpass[1] {
			u, err := getUser(userpass[0])
			if err != nil {
				u.Id = userpass[0]
				u.Type = "User"
			}
			return u
		}
	}
	return consolio.User{}
}

func md5string(i string) string {
	h := md5.New()
	h.Write([]byte(i))
	return hex.EncodeToString(h.Sum(nil))
}

func performAuth(w http.ResponseWriter, r *http.Request) {
	assertion := r.FormValue("assertion")
	if assertion == "" {
		showError(w, r, "No assertion requested.", 400)
		return
	}
	data := map[string]string{
		"assertion": assertion,
		"audience":  r.FormValue("audience"),
	}

	body, err := json.Marshal(&data)
	if err != nil {
		showError(w, r, "Error encoding request: "+err.Error(), 500)
		return
	}

	req, err := http.NewRequest("POST", BROWSERID_ENDPOINT,
		bytes.NewReader(body))
	if err != nil {
		panic(err)
	}

	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		showError(w, r, "Error transmitting request: "+err.Error(), 500)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		showError(w, r, "Invalid response code from browserid: "+res.Status, 500)
		return
	}

	resdata := browserIdData{}

	d := json.NewDecoder(res.Body)
	err = d.Decode(&resdata)
	if err != nil {
		showError(w, r, "Error decoding browserid response: "+err.Error(), 500)
		return
	}

	if resdata.Status != "okay" {
		showError(w, r, "Browserid status was not OK: "+
			resdata.Status+"/"+resdata.Reason, 500)
		return
	}

	if time.Now().Unix()*1000 >= int64(resdata.Expires) {
		glog.Warningf("browserId assertion had expired as of %v (current is %v)",
			resdata.Expires, time.Now().Unix())
		showError(w, r, "Browserid assertion is expired", 500)
		return
	}

	encoded, err := secureCookie.Encode("consolio.User", resdata)
	if err != nil {
		showError(w, r, "Couldn't encode cookie: "+err.Error(), 500)
		return
	}

	cookie := &http.Cookie{
		Name:  AUTH_COOKIE,
		Value: encoded,
		Path:  "/",
	}
	http.SetCookie(w, cookie)

	glog.Infof("Logged in %v", resdata.Email)

	mustEncode(w, map[string]interface{}{
		"email":    resdata.Email,
		"emailmd5": md5string(resdata.Email),
	})
}

func serveLogin(w http.ResponseWriter, r *http.Request) {
	me := whoami(r)

	if me.Id == "" {
		performAuth(w, r)
	} else {
		glog.Infof("Reusing existing thing: %v", me.Id)
		mustEncode(w, map[string]interface{}{
			"email":    me.Id,
			"emailmd5": md5string(me.Id),
			"prefs":    me.Prefs,
		})
	}
}

func serveLogout(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:  AUTH_COOKIE,
		Value: "",
		Path:  "/",
	}

	http.SetCookie(w, cookie)
}

func handleUserAuthToken(w http.ResponseWriter, r *http.Request) {
	me := whoami(r)
	// If the user doesn't have an auth token, make one.
	if me.AuthToken == "" {
		handleUpdateUserAuthToken(w, r)
		return
	}

	mustEncode(w, map[string]string{"token": me.AuthToken})
}

func handleUpdateUserAuthToken(w http.ResponseWriter, r *http.Request) {
	me := whoami(r)
	key := "u-" + me.Id
	user := consolio.User{}

	err := db.Update(key, 0, func(current []byte) ([]byte, error) {
		if len(current) > 0 {
			err := json.Unmarshal(current, &user)
			if err != nil {
				return nil, err
			}
		}

		// Common fields
		user.Id = me.Id
		user.Type = "user"

		user.AuthToken = randstring(16)

		return json.Marshal(user)
	})

	if err != nil {
		showError(w, r, err.Error(), 500)
		return
	}

	mustEncode(w, map[string]string{"token": user.AuthToken})
}
