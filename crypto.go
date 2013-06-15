package main

import (
	"bytes"
	"io"
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

var encryptKeys openpgp.EntityList

func encrypt(s string) string {
	buf := &bytes.Buffer{}

	wa, err := armor.Encode(buf, "PGP MESSAGE", nil)
	if err != nil {
		log.Fatalf("Can't make armor: %v", err)
	}

	w, err := openpgp.Encrypt(wa, encryptKeys, nil, nil, nil)
	if err != nil {
		log.Fatalf("Error encrypting: %v", err)
	}
	_, err = io.Copy(w, strings.NewReader(s))
	if err != nil {
		log.Fatalf("Error encrypting: %v", err)
	}
	w.Close()
	wa.Close()

	return buf.String()
}

func initPgp(kr string, keyids []string) {
	f, err := os.Open(kr)
	if err != nil {
		log.Fatalf("Can't open keyring: %v", err)
	}
	defer f.Close()

	kl, err := openpgp.ReadKeyRing(f)
	if err != nil {
		log.Fatalf("Can't read keyring: %v", err)
	}

	for _, w := range keyids {
		for _, e := range kl {
			if e.PrimaryKey.KeyIdShortString() == w {
				encryptKeys = append(encryptKeys, e)
			}
		}
	}

	if len(encryptKeys) != len(keyids) {
		log.Fatalf("Couldn't find all keys")
	}
}
