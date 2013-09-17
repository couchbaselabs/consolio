package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/couchbaselabs/consolio/tools"
)

var (
	keyRingPath = flag.String("keyring", "", "Your secret keyring")
	keyPassword = flag.String("password", "", "Crypto password")
	bindAddr    = flag.String("bind", ":8475", "Binding address")
	confType    = flag.String("type", "sgw", "Configuration type (sgw|cbgb)")
)

var obGens = map[string]func() interface{}{}

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
	log.SetFlags(0)

	if _, ok := obGens[*confType]; !ok {
		log.Fatalf("Invalid conf type: %v", *confType)
	}

	err := consoliotools.InitCrypto(*keyRingPath, *keyPassword)
	if err != nil {
		log.Fatalf("Error initializing crypto: %v", err)
	}

	baseUrl := flag.Arg(0)
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		u := baseUrl + req.URL.Path[1:]
		db, err := getit(u)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}

		d, err := json.Marshal(db)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(d)
	})

	server := &http.Server{
		Addr:         *bindAddr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: time.Second,
	}

	server.ListenAndServe()
}
