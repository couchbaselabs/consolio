package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var staticPath = flag.String("static", "static", "Path to the static content")

func main() {
	addr := flag.String("addr", ":8675", "http listen address")
	flag.Parse()

	r := mux.NewRouter()

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*staticPath))))

	http.Handle("/", r)

	log.Printf("Listening on %v", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
