package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"

	"github.com/couchbaselabs/go-couchbase"
	"github.com/gorilla/mux"
)

var staticPath = flag.String("static", "static", "Path to the static content")

var db *couchbase.Bucket

func showError(w http.ResponseWriter, r *http.Request,
	msg string, code int) {
	log.Printf("Reporting error %v/%v", code, msg)
	http.Error(w, msg, code)
}

func mustEncode(w io.Writer, i interface{}) {
	if headered, ok := w.(http.ResponseWriter); ok {
		headered.Header().Set("Cache-Control", "no-cache")
		headered.Header().Set("Content-type", "application/json")
	}

	e := json.NewEncoder(w)
	if err := e.Encode(i); err != nil {
		panic(err)
	}
}

func RewriteURL(to string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = to
		h.ServeHTTP(w, r)
	})
}

func main() {
	addr := flag.String("addr", ":8675", "http listen address")
	cbServ := flag.String("couchbase", "http://localhost:8091/",
		"URL to couchbase")
	cbBucket := flag.String("bucket", "consolio", "couchbase bucket")
	secCookKey := flag.String("cookieKey", "thespywholovedme",
		"The secure cookie auth code.")
	flag.Parse()

	r := mux.NewRouter()

	r.HandleFunc("/auth/login", serveLogin).Methods("POST")
	r.HandleFunc("/auth/logout", serveLogout).Methods("POST")

	// application pages
	appPages := []string{
		"/index/",
	}

	for _, p := range appPages {
		r.PathPrefix(p).Handler(RewriteURL("app.html",
			http.FileServer(http.Dir(*staticPath))))
	}

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*staticPath))))

	initSecureCookie([]byte(*secCookKey))

	http.Handle("/", r)

	var err error
	db, err = dbConnect(*cbServ, *cbBucket)
	if err != nil {
		log.Fatalf("Error connecting to couchbase: %v", err)
	}

	log.Printf("Listening on %v", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
