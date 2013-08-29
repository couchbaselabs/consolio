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
	confType    = flag.String("type", "sgw", "Configuration type (sgw|cbgb)")
)

type SyncGateway struct {
	Extra struct {
		Sync, Users    *json.RawMessage
		DBName, Server string
		Pass           string `json:"db_pass"`
	}
}

func (s *SyncGateway) MarshalJSON() ([]byte, error) {
	d := s.Extra
	m := map[string]interface{}{}
	m["bucket"] = d.DBName
	m["sync"] = d.Sync
	if d.Users != nil {
		m["users"] = d.Users
	}

	u, err := url.Parse(d.Server)
	if err == nil {
		pass, err := consoliotools.Decrypt(d.Pass)
		if err == nil {
			u.User = url.UserPassword(d.DBName, pass)
		} else {
			log.Printf("Error decrypting password: %v", err)
		}
		m["server"] = u.String()
	} else {
		m["server"] = d.Server
	}

	return json.Marshal(m)
}

type Database struct {
	Password string
}

func (d *Database) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"memoryOnly":       0,
		"numPartitions":    1,
		"passwordHash":     "",
		"passwordHashFunc": "",
		"passwordSalt":     "",
		"quotaBytes":       500 * 1024 * 1024,
	}

	pass, err := consoliotools.Decrypt(d.Password)
	if err == nil {
		m["passwordHash"] = pass
	} else {
		log.Printf("Error decrypting password: %v", err)
	}

	return json.Marshal(m)
}

var obGens = map[string]func() interface{}{
	"sgw":  func() interface{} { return &SyncGateway{} },
	"cbgb": func() interface{} { return &Database{} },
}

func getit(u string) (interface{}, error) {
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

	db := obGens[*confType]()
	err = d.Decode(db)
	return db, nil
}

func main() {
	flag.Parse()

	if _, ok := obGens[*confType]; !ok {
		log.Fatalf("Invalid conf type: %v", *confType)
	}

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
