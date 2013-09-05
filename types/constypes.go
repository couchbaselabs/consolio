package consolio

import (
	"encoding/json"
	"fmt"
	"time"
)

type Item struct {
	Name      string                 `json:"name"`
	Password  string                 `json:"password,omitempty"`
	Type      string                 `json:"type"`
	Owner     string                 `json:"owner"`
	Enabled   bool                   `json:"enabled"`
	LastMod   time.Time              `json:"lastmod"`
	ExtraInfo map[string]interface{} `json:"extra"`

	// Stuff provided by the service
	URL        string           `json:"url"`
	LastChange time.Time        `json:"last_state_change"`
	State      string           `json:"state"`
	LastStat   time.Time        `json:"last_stats"`
	Stats      *json.RawMessage `json:"stats"`
}

func (i Item) String() string {
	return fmt.Sprintf("%v:%v", i.Type, i.Name)
}

type ChangeEvent struct {
	Type      string    `json:"type"`
	Item      Item      `json:"item"`
	Timestamp time.Time `json:"ts"`
	Processed time.Time `json:"processed"`
	Error     string    `json:"error,omitempty"`
	Failures  int       `json:"failures"`

	ID string
}

func (c ChangeEvent) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"type":     c.Type,
		"item":     c.Item,
		"ts":       c.Timestamp,
		"failures": c.Failures,
	}

	if !c.Processed.IsZero() {
		m["processed"] = c.Processed
	}
	if c.ID != "" {
		m["id"] = c.ID
	}

	return json.Marshal(m)
}

type User struct {
	Id        string                 `json:"id"`
	Type      string                 `json:"type"`
	Admin     bool                   `json:"admin"`
	AuthToken string                 `json:"auth_token,omitmepty"`
	Internal  bool                   `json:"internal"`
	Prefs     map[string]interface{} `json:"prefs"`
}
