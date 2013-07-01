package consolio

import (
	"encoding/json"
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
	Size int64  `json:"size"`
	URL  string `json:"url"`
}

type ChangeEvent struct {
	Type      string    `json:"type"`
	Item      Item      `json:"item"`
	Timestamp time.Time `json:"ts"`
	Processed time.Time `json:"processed"`

	ID string
}

func (c ChangeEvent) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"type": c.Type,
		"item": c.Item,
		"ts":   c.Timestamp,
	}

	if !c.Processed.IsZero() {
		m["processed"] = c.Processed
	}
	if c.ID != "" {
		m["id"] = c.ID
	}

	return json.Marshal(m)
}
