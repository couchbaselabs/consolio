package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"github.com/couchbaselabs/consolio/tools"
)

func init() {
	obGens["sgw"] = func() interface{} { return &SyncGateway{} }
}

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
	if d.Sync == nil {
		return nil, fmt.Errorf("Invalid JSON, missing syncgw")
	}
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
