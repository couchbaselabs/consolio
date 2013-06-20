package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	_ "crypto/ecdsa"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"

	"code.google.com/p/go.crypto/openpgp"
	"code.google.com/p/go.crypto/openpgp/armor"

	"github.com/couchbaselabs/consolio/types"
)

var (
	todoUrl     = flag.String("url", "", "URL to TODO API")
	keyRingPath = flag.String("keyring", "", "Your secret keyring")
	keyPassword = flag.String("password", "", "Crypto password")
)

var keys openpgp.EntityList

func initCrypto() {
	f, err := os.Open(*keyRingPath)
	if err != nil {
		log.Fatalf("Can't open keyring: %v", err)
	}
	defer f.Close()

	keys, err = openpgp.ReadKeyRing(f)
	if err != nil {
		log.Fatalf("Can't read keyring: %v", err)
	}
}

func decrypt(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	raw, err := armor.Decode(strings.NewReader(s))
	if err != nil {
		return "", err
	}

	d, err := openpgp.ReadMessage(raw.Body, keys,
		func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
			kp := []byte(*keyPassword)
			if symmetric {
				return kp, nil
			}
			for _, k := range keys {
				err := k.PrivateKey.Decrypt(kp)
				if err == nil {
					return nil, nil
				}
			}
			return nil, fmt.Errorf("No key")
		},
		nil)
	if err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadAll(d.UnverifiedBody)
	return string(bytes), err
}

func markDone(id string) error {
	u := *todoUrl + id
	res, err := http.Post(u, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 204 {
		return fmt.Errorf("HTTP error marking task done: %v", res.Status)
	}

	return nil
}

func processTodo() error {
	log.Printf("Processing TODOs...")
	res, err := http.Get(*todoUrl)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("Bad HTTP response: %v", res.Status)
	}

	data := []consolio.ChangeEvent{}
	d := json.NewDecoder(res.Body)
	err = d.Decode(&data)
	if err != nil {
		return err
	}

	for _, e := range data {
		pw, err := decrypt(e.Database.Password)
		if err != nil {
			return err
		}
		log.Printf("Found %v -> %v %v - %v",
			e.ID, e.Type, e.Database.Name, pw)

		err = markDone(e.ID)
		if err != nil {
			return err
		}
	}

	return nil
}