package consoliotools

import (
	"fmt"
	"io/ioutil"
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
	password string
	keys     openpgp.EntityList
)

func InitCrypto(keyRingPath, pass string) error {
	f, err := os.Open(keyRingPath)
	if err != nil {
		return err
	}
	defer f.Close()

	keys, err = openpgp.ReadKeyRing(f)
	if err != nil {
		return err
	}

	password = pass
	return nil
}

func Decrypt(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	raw, err := armor.Decode(strings.NewReader(s))
	if err != nil {
		return "", err
	}

	d, err := openpgp.ReadMessage(raw.Body, keys,
		func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
			kp := []byte(password)
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
