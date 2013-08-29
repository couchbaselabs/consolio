package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/couchbaselabs/consolio/tools"
)

var (
	keyRingPath = flag.String("keyring", "", "Your secret keyring")
	keyPassword = flag.String("password", "", "Crypto password")
	bindAddr    = flag.String("bind", ":8475", "Binding address")
)

type Database struct {
	Sync, Users  *json.RawMessage
	Name, Server string
	Pass         string `json:"db_pass"`
}

func (d Database) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	m["bucket"] = d.Name
	m["sync"] = d.Sync
	if d.Users != nil {
		m["users"] = d.Users
	}

	u, err := url.Parse(d.Server)
	if err == nil {
		pass, err := consoliotools.Decrypt(d.Pass)
		if err == nil {
			u.User = url.UserPassword(d.Name, pass)
		} else {
			log.Printf("Error decrypting password: %v", err)
		}
		m["server"] = u.String()
	} else {
		m["server"] = d.Server
	}

	return json.Marshal(m)
}

func getit(u string) (*Database, error) {
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error getting config: %v", res.Status)
	}

	d := json.NewDecoder(res.Body)
	d.UseNumber()

	reson := struct {
		Name  string
		Extra Database
	}{}
	err = d.Decode(&reson)
	if err == nil {
		reson.Extra.Name = reson.Name
	}
	return &reson.Extra, nil
}

func main() {
	flag.Parse()

	consoliotools.InitCrypto(*keyRingPath, *keyPassword)
	baseUrl := flag.Arg(0)
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		u := baseUrl + req.URL.Path[1:]
		db, err := getit(u)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		e := json.NewEncoder(w)
		e.Encode(db)
	})

	http.ListenAndServe(*bindAddr, nil)
}
