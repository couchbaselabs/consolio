package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/couchbaselabs/go-couchbase"
	"github.com/dustin/gomemcached"
	"github.com/gorilla/mux"
)

var staticPath = flag.String("static", "static", "Path to the static content")

var db *couchbase.Bucket

var eventCh = make(chan HookEvent, 10)

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

	eventCh <- HookEvent{"create", d}

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
		if r.Doc.Json != nil {
			rv = append(rv, r.Doc.Json)
		}
	}

	mustEncode(w, rv)
}

func handleGetDB(w http.ResponseWriter, req *http.Request) {
	d := Database{}
	err := db.Get("db-"+mux.Vars(req)["name"], &d)
	switch {
	case gomemcached.IsNotFound(err):
		showError(w, req, "Not found", 404)
	case err != nil:
		showError(w, req, err.Error(), 500)
	case d.Type != "database":
		showError(w, req, "Incorrect type", 400)
	case d.Owner != whoami(req).Id:
		showError(w, req, "Not your DB", 403)
	default:
		d.Password = ""
		mustEncode(w, d)
	}
}

func handleDeleteDB(w http.ResponseWriter, req *http.Request) {
	d := Database{}
	k := "db-" + mux.Vars(req)["name"]
	err := db.Get(k, &d)
	switch {
	case gomemcached.IsNotFound(err):
		showError(w, req, "Not found", 404)
	case err != nil:
		showError(w, req, err.Error(), 500)
	case d.Type != "database":
		showError(w, req, "Incorrect type", 400)
	case d.Owner != whoami(req).Id:
		showError(w, req, "Not your DB", 403)
	}

	err = db.Delete(k)
	if err != nil {
		showError(w, req, err.Error(), 500)
		return
	}

	eventCh <- HookEvent{"delete", d}

	w.WriteHeader(204)
}

func getWebhooks() ([]Webhook, error) {
	rv := []Webhook{}

	viewRes := struct {
		Rows []struct {
			Key, Value string
		}
	}{}

	err := db.ViewCustom("consolio", "webhooks", nil, &viewRes)
	if err != nil {
		return rv, err
	}

	for _, r := range viewRes.Rows {
		rv = append(rv, Webhook{Name: r.Key, Url: r.Value, Type: "webhook"})
	}

	return rv, err
}

func handleListWebhooks(w http.ResponseWriter, req *http.Request) {
	hooks, err := getWebhooks()
	if err != nil {
		showError(w, req, err.Error(), 500)
		return
	}

	mustEncode(w, hooks)
}

func adminRequired(r *http.Request, rm *mux.RouteMatch) bool {
	return whoami(r).Admin
}

func handleNewWebhook(w http.ResponseWriter, req *http.Request) {
	wh := Webhook{
		Name: req.FormValue("name"),
		Url:  req.FormValue("url"),
		Type: "webhook",
	}

	_, err := url.Parse(wh.Url)
	if err != nil {
		showError(w, req, err.Error(), 500)
		return
	}

	k := "wh-" + wh.Name
	err = db.Set(k, 0, wh)
	if err != nil {
		showError(w, req, err.Error(), 500)
		return
	}

	mustEncode(w, wh)
}

func handleDeleteWebhook(w http.ResponseWriter, req *http.Request) {
	k := "wh-" + mux.Vars(req)["name"]
	err := db.Delete(k)
	switch {
	case gomemcached.IsNotFound(err):
		showError(w, req, "Not found", 404)
	case err != nil:
		showError(w, req, err.Error(), 500)
	default:
		w.WriteHeader(204)
	}
}

func handleMe(w http.ResponseWriter, req *http.Request) {
	mustEncode(w, whoami(req))
}

func RewriteURL(to string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = to
		h.ServeHTTP(w, r)
	})
}

func runHook(wh Webhook, content []byte) error {
	req, err := http.NewRequest("POST", wh.Url, bytes.NewReader(content))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("HTTP Error: %v", res.Status)
	}
	return nil
}

func runHooks(h HookEvent) {
	hooks, err := getWebhooks()
	if err != nil {
		log.Printf("Error getting web hooks:  %v", err)
		return
	}

	content, err := json.Marshal(h)
	if err != nil {
		log.Printf("Error marshaling hook event: %v", err)
		return
	}

	for _, wh := range hooks {
		err := runHook(wh, content)
		if err != nil {
			log.Printf("Error running hook %v -> %v: %v", h, wh, err)
		}
	}
}

func hookRunner() {
	for h := range eventCh {
		runHooks(h)
	}
}

func main() {
	addr := flag.String("addr", ":8675", "http listen address")
	cbServ := flag.String("couchbase", "http://localhost:8091/",
		"URL to couchbase")
	cbBucket := flag.String("bucket", "consolio", "couchbase bucket")
	secCookKey := flag.String("cookieKey", "thespywholovedme",
		"The secure cookie auth code.")
	flag.Parse()

	go hookRunner()

	r := mux.NewRouter()

	r.HandleFunc("/auth/login", serveLogin).Methods("POST")
	r.HandleFunc("/auth/logout", serveLogout).Methods("POST")

	// application pages
	appPages := []string{
		"/index/",
		"/db/",
		"/admin/",
	}

	for _, p := range appPages {
		r.PathPrefix(p).Handler(RewriteURL("app.html",
			http.FileServer(http.Dir(*staticPath))))
	}

	r.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir(*staticPath))))

	r.HandleFunc("/api/database/{name}/", handleGetDB).Methods("GET")
	r.HandleFunc("/api/database/{name}/", handleDeleteDB).Methods("DELETE")
	r.HandleFunc("/api/database/", handleListDBs).Methods("GET")
	r.HandleFunc("/api/database/", handleNewDB).Methods("POST")
	r.HandleFunc("/api/me/", handleMe).Methods("GET")

	r.HandleFunc("/api/webhook/",
		handleListWebhooks).Methods("GET").MatcherFunc(adminRequired)
	r.HandleFunc("/api/webhook/",
		handleNewWebhook).Methods("POST").MatcherFunc(adminRequired)
	r.HandleFunc("/api/webhook/{name}/",
		handleDeleteWebhook).Methods("DELETE").MatcherFunc(adminRequired)

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
