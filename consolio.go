package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"

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

func isValidDBName(n string) bool {
	return true
}

func handleNewDB(w http.ResponseWriter, req *http.Request) {
	d := Database{
		Name:    strings.TrimSpace(req.FormValue("name")),
		Type:    "database",
		Owner:   whoami(req).Id,
		Enabled: true,
	}

	if !isValidDBName(d.Name) {
		showError(w, req, "Invalid DB Name", 400)
		return
	}

	added, err := db.Add("db-"+d.Name, 0, d)
	if err != nil {
		showError(w, req, "Error adding to DB: "+err.Error(), 500)
		return
	}
	if !added {
		showError(w, req, "Did not add to DB (no error)", 500)
		return
	}

	mustEncode(w, d)
}

func handleListDBs(w http.ResponseWriter, req *http.Request) {
	viewRes := struct {
		Rows []struct {
			Doc struct {
				Json *json.RawMessage
			}
		}
	}{}

	me := whoami(req).Id

	empty := &json.RawMessage{'{', '}'}
	err := db.ViewCustom("consolio", "databases",
		map[string]interface{}{
			"reduce":       false,
			"include_docs": true,
			"start_key":    []interface{}{me},
			"end_key":      []interface{}{me, empty},
		},
		&viewRes)
	if err != nil {
		showError(w, req, "Did Error listing stuff: "+
			err.Error(), 500)
		return
	}

	rv := []interface{}{}
	for _, r := range viewRes.Rows {
		rv = append(rv, r.Doc.Json)
	}

	mustEncode(w, rv)
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
		"/db/",
	}

	for _, p := range appPages {
		r.PathPrefix(p).Handler(RewriteURL("app.html",
			http.FileServer(http.Dir(*staticPath))))
	}

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*staticPath))))

	r.HandleFunc("/api/database/", handleListDBs).Methods("GET")
	r.HandleFunc("/api/database/", handleNewDB).Methods("POST")

	r.Handle("/", http.RedirectHandler("/index/", 302))

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
