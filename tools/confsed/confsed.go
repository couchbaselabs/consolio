package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type handler struct {
	dest url.URL
}

var rewriter *strings.Replacer

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	du := h.dest
	du.Path = req.URL.Path
	du.RawQuery = req.URL.RawQuery
	req.URL = &du
	req.RequestURI = ""

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer res.Body.Close()

	for k, vs := range res.Header {
		h := w.Header()
		h.Del(k)
		for _, v := range vs {
			h.Add(k, v)
		}
	}
	w.WriteHeader(res.StatusCode)

	if strings.Contains(res.Header.Get("content-type"), "json") {
		io.Copy(w, rewriteJson(res.Body, rewriter.Replace))
	} else {
		io.Copy(w, res.Body)
	}
}

func initRewrite(conf string) {
	if conf == "" {
		rewriter = strings.NewReplacer()
		return
	}

	f, err := os.Open(conf)
	if err != nil {
		log.Fatalf("Error opening %v: %v", conf, err)
	}
	defer f.Close()
	d := json.NewDecoder(f)
	m := map[string]string{}
	err = d.Decode(&m)
	if err != nil {
		log.Fatalf("Error parsing %v: %v", conf, err)
	}

	params := []string{}
	for k, v := range m {
		params = append(params, k, v)
	}

	rewriter = strings.NewReplacer(params...)
}

func main() {
	bindAddr := flag.String("bind", ":7081", "Address to listen")
	rewriteConf := flag.String("rewriteconf", "",
		"Path to json rewrite rules")

	flag.Parse()

	if flag.NArg() != 1 {
		log.Fatalf("Where to, sir?")
	}

	initRewrite(*rewriteConf)

	u, err := url.Parse(flag.Arg(0))
	if err != nil {
		log.Fatalf("Error parsing url: %v", err)
	}

	log.Fatal(http.ListenAndServe(*bindAddr, &handler{*u}))
}
