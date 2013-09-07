package main

import (
	"encoding/json"
	"log"

	"github.com/couchbaselabs/consolio/tools"
)

func init() {
	obGens["cbgb"] = func() interface{} { return &Database{} }
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
