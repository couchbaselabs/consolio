package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	_ "crypto/ecdsa"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"

	"code.google.com/p/go.crypto/openpgp"
	"code.google.com/p/go.crypto/openpgp/armor"
)

var (
	keyRingPath = flag.String("keyring", "", "Your secret keyring")
	keyPassword = flag.String("password", "", "Crypto password")

	keys openpgp.EntityList
)

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
