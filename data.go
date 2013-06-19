package main

import (
	"encoding/json"
	"time"
)

type Database struct {
	Name     string    `json:"name"`
	Password string    `json:"password,omitempty"`
	Type     string    `json:"type"`
	Owner    string    `json:"owner"`
	Enabled  bool      `json:"enabled"`
	LastMod  time.Time `json:"lastmod"`
	Size     int64     `json:"size"`
}

type Webhook struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Type string `json:"type"`
}

type ChangeEvent struct {
	Type      string    `json:"type"`
	Database  Database  `json:"database"`
	Timestamp time.Time `json:"ts"`
	Processed time.Time `json:"processed"`

	ID string
}

func (c ChangeEvent) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"type":     c.Type,
		"database": c.Database,
		"ts":       c.Timestamp,
	}

	if !c.Processed.IsZero() {
		m["processed"] = c.Processed
	}
	if c.ID != "" {
		m["id"] = c.ID
	}

	return json.Marshal(m)
}
