package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var proxyBind = flag.String("proxybind", ":8083", "HTTP listen address")
var targetUrl = flag.String("syncurl", "http://localhost:4985/",
	"sync gateway admin URL")

var proxyTarget *url.URL
var brokenHost string

func proxyAllow(req *http.Request) bool {
	return true
}

func direct(req *http.Request) {
	if proxyAllow(req) {
		req.URL.Scheme = proxyTarget.Scheme
		req.URL.Host = proxyTarget.Host
	} else {
		req.URL.Scheme = "http"
		req.URL.Host = brokenHost
	}
}

type errorizer struct{}

func (e errorizer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "NO", 403)
}

func sgwProxy() {
	if *proxyBind == "" || *targetUrl == "" {
		return
	}

	u, err := url.Parse(*targetUrl)
	if err != nil {
		log.Printf("Error parsing target URL: %v", err)
	}
	proxyTarget = u

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

	log.Fatal(http.ListenAndServe(*proxyBind, proxy))
}
