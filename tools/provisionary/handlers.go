package main

import (
	"log"

	"github.com/couchbaselabs/consolio/types"
)

type handler func(consolio.ChangeEvent, string) error

var handlers []handler

func initHandlers() {
	handlers = append(handlers, logHandler)
}

func logHandler(e consolio.ChangeEvent, pw string) error {
	log.Printf("Found %v -> %v %v - %q",
		e.ID, e.Type, e.Database.Name, pw)
	return nil
}
