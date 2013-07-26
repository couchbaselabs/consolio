package main

import (
	"flag"
	"net/http"
	"time"

	"github.com/couchbaselabs/consolio/tools"
	"github.com/golang/glog"
)

var (
	addr     = flag.String("bindaddr", ":8555", "HTTP bind address")
	pollFreq = flag.Duration("pollfreq", time.Minute*5,
		"How frequently to run failsafe polling")
	keyRingPath = flag.String("keyring", "", "Your secret keyring")
	keyPassword = flag.String("password", "", "Crypto password")
)

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
			glog.Infof("Error processing things: %v", err)
		}
	}
}

func maybefire() {
	select {
	case hookCh <- true:
	default:
	}
}

func main() {
	flag.Parse()

	consoliotools.InitCrypto(*keyRingPath, *keyPassword)
	initHandlers()

	go provisionLoop()

	maybefire()

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		maybefire()
		w.WriteHeader(202)
	})

	glog.Fatal(http.ListenAndServe(*addr, nil))
}
