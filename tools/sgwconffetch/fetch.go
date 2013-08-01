package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/couchbaselabs/consolio/tools"
	"net/url"
)

var (
	keyRingPath = flag.String("keyring", "", "Your secret keyring")
	keyPassword = flag.String("password", "", "Crypto password")
)

type Database struct {
	Sync, Users    *json.RawMessage
	Bucket, Server string
	Pass           string `json:"db_pass"`
}

func (d Database) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	m["bucket"] = d.Bucket
	m["sync"] = d.Sync
	if d.Users != nil {
		m["users"] = d.Users
	}

	u, err := url.Parse(d.Server)
	if err == nil {
		pass, err := consoliotools.Decrypt(d.Pass)
		if err == nil {
			u.User = url.UserPassword(d.Bucket, pass)
		} else {
			log.Printf("Error decrypting password: %v", err)
		}
		m["server"] = u.String()
	} else {
		m["server"] = d.Server
	}

	return json.Marshal(m)
}

type Config struct {
	AdminInterface *json.RawMessage `json:"adminInterface"`
	Interface      *json.RawMessage `json:"interface"`
	Persona        *json.RawMessage `json:"persona"`
	Log            *json.RawMessage `json:"log"`
	Databases      map[string]Database
}

func getit(u string) error {
	res, err := http.Get(flag.Arg(0))
	if err != nil {
		log.Fatalf("Error getting config: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("HTTP error getting config: %v", res.Status)
	}

	d := json.NewDecoder(res.Body)
	d.UseNumber()

	conf := Config{}
	err = d.Decode(&conf)
	if err != nil {
		log.Fatalf("Error decoding config: %v", err)
	}

	e := json.NewEncoder(os.Stdout)
	return e.Encode(&conf)
}

func main() {
	flag.Parse()

	consoliotools.InitCrypto(*keyRingPath, *keyPassword)

	getit(flag.Arg(0))
}
