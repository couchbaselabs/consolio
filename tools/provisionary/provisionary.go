package main

import (
	"flag"
	"log"
	"net/http"
	"time"
)

var addr = flag.String("bindaddr", ":8555", "HTTP bind address")
var pollFreq = flag.Duration("pollfreq", time.Minute*5,
	"How frequently to run failsafe polling")

var hookCh = make(chan bool, 1)

func provisionLoop() {
	t := time.Tick(*pollFreq)
	for {
		select {
		case <-t:
		case <-hookCh:
		}
		err := processTodo()
		if err != nil {
			log.Printf("Error processing things: %v", err)
		}
	}
}

func main() {
	flag.Parse()

	initCrypto()

	go provisionLoop()

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		select {
		case hookCh <- true:
		default:
		}
		w.WriteHeader(202)
	})

	log.Fatal(http.ListenAndServe(*addr, nil))
}
