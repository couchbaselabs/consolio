package main

type Database struct {
	Name     string    `json:"name"`
	Password string    `json:"password"`
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

type HookEvent struct {
	Type     string   `json:"type"`
	Database Database `json:"database"`
}
