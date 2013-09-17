package main

import (
	"flag"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/couchbaselabs/consolio/types"
)

var proxyBind = flag.String("proxybind", ":8083", "HTTP listen address")
var targetUrl = flag.String("syncurl", "http://localhost:4985/",
	"sync gateway admin URL")
var proxyAllowAdmin = flag.Bool("proxyadmin", false, "Proxy all admin reqs")

var proxyTarget *url.URL
var brokenHost string

func proxyAllow(req *http.Request) bool {
	u := whoami(req)
	if *proxyAllowAdmin && u.Admin {
		return true
	}

	parts := strings.Split(req.URL.Path, "/")
	if len(parts) < 2 {
		return false
	}
	name := parts[1]

	d := consolio.Item{}
	err := db.Get("sgw-"+name, &d)
	return err == nil && d.Owner == u.Id
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
		glog.Infof("Error parsing target URL: %v", err)
	}
	proxyTarget = u

	proxy := &httputil.ReverseProxy{Director: direct}

	a, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		glog.Fatalf("Can't resolve address: %v", err)
	}
	errorist, err := net.ListenTCP("tcp", a)
	if err != nil {
		glog.Fatalf("Error creating errorist: %v", err)
	}

	go http.Serve(errorist, errorizer{})

	brokenHost = errorist.Addr().String()
	glog.Infof("Errorizer is on %v", brokenHost)

	glog.Infof("Running sgw proxy on %v", *proxyBind)
	server := &http.Server{
		Addr:        *proxyBind,
		Handler:     proxy,
		ReadTimeout: 30 * time.Second,
	}
	glog.Fatal(server.ListenAndServe())
}
