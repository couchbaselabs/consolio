package main

type Database struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Type     string `json:"type"`
	Owner    string `json:"owner"`
	Enabled  bool   `json:"enabled"`
	Size     int64  `json:"size"`
}
