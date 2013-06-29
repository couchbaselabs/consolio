package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var bindAddr = flag.String("bind", ":8083", "HTTP listen address")
var targetUrl = flag.String("dest", "http://localhost:4985/", "Target DB.")

var target *url.URL
var brokenHost string

func direct(req *http.Request) {
	permitted := true
	if permitted {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
	} else {
		req.URL.Scheme = "http"
		req.URL.Host = brokenHost
	}
}

type errorizer struct{}

func (e errorizer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "NO", 403)
}

func main() {
	flag.Parse()

	u, err := url.Parse(*targetUrl)
	if err != nil {
		log.Printf("Error parsing target URL: %v", err)
	}
	target = u

	proxy := &httputil.ReverseProxy{Director: direct}

	a, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("Can't resolve address: %v", err)
	}
	errorist, err := net.ListenTCP("tcp", a)
	if err != nil {
		log.Fatalf("Error creating errorist: %v", err)
	}
	go http.Serve(errorist, errorizer{})

	brokenHost = errorist.Addr().String()
	log.Printf("Errorizer is on %v", brokenHost)

	log.Fatal(http.ListenAndServe(*bindAddr, proxy))
}
