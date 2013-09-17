package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/couchbaselabs/go-couchbase"
	"github.com/dustin/gomemcached"
	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"github.com/couchbaselabs/consolio/types"
)

const maxFailures = 5

var staticPath = flag.String("static", "static", "Path to the static content")
var backendPrefix = flag.String("backendPrefix", "/backend/",
	"HTTP path prefix for backend API")

var db *couchbase.Bucket

var eventCh = make(chan consolio.ChangeEvent, 10)

func showError(w http.ResponseWriter, r *http.Request,
	msg string, code int) {
	glog.Infof("Reporting error %v/%v", code, msg)
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

var dbValidator = regexp.MustCompile(`^([-+=/_.@\p{L}\p{Nd}]+|\*)$`)

func isValidDBName(n string) bool {
	return len(n) > 0 && n[0] != '_' && dbValidator.MatchString(n)
}

func handleNewDB(w http.ResponseWriter, req *http.Request) {
	d := consolio.Item{
		Name:     strings.TrimSpace(req.FormValue("name")),
		Password: encrypt(strings.TrimSpace(req.FormValue("password"))),
		Type:     "database",
		Owner:    whoami(req).Id,
		Enabled:  true,
		LastMod:  time.Now().UTC(),
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

	err = recordEvent("create", d)
	if err != nil {
		showError(w, req, "Did not record mutation event: "+err.Error(), 500)
		return
	}

	mustEncode(w, d)
}

func handleMkDBConf(w http.ResponseWriter, req *http.Request) {
	name := mux.Vars(req)["name"]
	d := consolio.Item{}
	err := db.Get("db-"+name, &d)
	if err != nil {
		glog.Warningf("Error retrieving database %q: %v", name, err)
		showError(w, req, "Unknown database", 404)
		return
	}

	if !d.Enabled {
		glog.Warningf("Trying to activate disabled DB %q", name)
		showError(w, req, "Disabled database", 404)
		return
	}

	mustEncode(w, d)
}

func tstr(t time.Time) string {
	return t.Format("20060102150405.999999999")
}

func hashstr(s string) string {
	h := sha1.New()
	_, err := h.Write([]byte(s))
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func recordEvent(t string, i consolio.Item) error {
	ts := time.Now().UTC()
	k := "ch-" + t + "-" + tstr(ts) + "-" + hashstr(i.Name)[:8]
	ev := consolio.ChangeEvent{Type: t, Item: i, Timestamp: ts}
	a, err := db.Add(k, 0, ev)
	if err != nil {
		return err
	}
	if !a {
		return fmt.Errorf("Failed to add %v", k)
	}
	eventCh <- ev
	return nil
}

func listItem(w http.ResponseWriter, req *http.Request, t string) {
	viewRes := struct {
		Rows []struct {
			Doc struct {
				Json *json.RawMessage
			}
		}
	}{}

	me := whoami(req).Id

	empty := &json.RawMessage{'{', '}'}
	err := db.ViewCustom("consolio", "items",
		map[string]interface{}{
			"reduce":       false,
			"include_docs": true,
			"stale":        false,
			"start_key":    []interface{}{t, me},
			"end_key":      []interface{}{t, me, empty},
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

func handleListDBs(w http.ResponseWriter, req *http.Request) {
	listItem(w, req, "database")
}

func handleGetDB(w http.ResponseWriter, req *http.Request) {
	d := consolio.Item{}
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
		mustEncode(w, d)
	}
}

func handleDeleteDB(w http.ResponseWriter, req *http.Request) {
	d := consolio.Item{}
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

	d.Password = ""
	err = recordEvent("delete", d)
	if err != nil {
		showError(w, req, "Did not record mutation event: "+err.Error(), 500)
		return
	}

	w.WriteHeader(204)
}

func handleListTopDBs(w http.ResponseWriter, req *http.Request) {
	viewRes := struct {
		Rows []struct {
			Key   float64
			Value struct {
				Name, Owner string
			}
		}
	}{}

	err := db.ViewCustom("consolio", "bysize", map[string]interface{}{
		"limit":      50,
		"descending": true,
		"stale":      false,
	}, &viewRes)
	if err != nil {
		showError(w, req, err.Error(), 500)
		return
	}

	type T struct {
		Size        float64
		Name, Owner string
	}
	res := []T{}
	for _, r := range viewRes.Rows {
		res = append(res, T{r.Key, r.Value.Name, r.Value.Owner})
	}

	mustEncode(w, res)
}

func adminRequired(r *http.Request, rm *mux.RouteMatch) bool {
	return whoami(r).Admin
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

func main() {
	addr := flag.String("addr", ":8675", "http listen address")
	cbServ := flag.String("couchbase", "http://localhost:8091/",
		"URL to couchbase")
	cbBucket := flag.String("bucket", "consolio", "couchbase bucket")
	secCookKey := flag.String("cookieKey", "thespywholovedme",
		"The secure cookie auth code.")
	keyRing := flag.String("keyring", "", "pgp keyring")
	encryptTo := flag.String("encryptTo", "",
		"pgp IDs for password recipients (comma separated)")
	flag.Parse()

	initPgp(*keyRing, strings.Split(*encryptTo, ","))

	go eventListener()

	r := mux.NewRouter()

	r.HandleFunc("/auth/login", serveLogin).Methods("POST")
	r.HandleFunc("/auth/logout", serveLogout).Methods("POST")

	// application pages
	appPages := []string{
		"/index/",
		"/db/",
		"/sgw/",
		"/admin/",
		"/terms_of_service/",
		"/acceptable_use/",
		"/privacy_policy/",
		"/dashboard/",
		"/faq/",
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

	r.HandleFunc("/api/sgw/{name}/", handleGetSGW).Methods("GET")       // Get SGW
	r.HandleFunc("/api/sgw/{name}/", handleDeleteSGW).Methods("DELETE") // Delete SGW
	r.HandleFunc("/api/sgw/{name}/", handleUpdateSGW).Methods("POST")   // Update the sync funk
	r.HandleFunc("/api/sgw/", handleListSGWs).Methods("GET")            // Get SGW List
	r.HandleFunc("/api/sgw/", handleNewSGW).Methods("POST")             // Create New SGW

	r.HandleFunc("/api/me/", handleMe).Methods("GET")
	r.HandleFunc("/api/me/token/", handleUserAuthToken).Methods("GET")

	r.HandleFunc("/api/topdbs/",
		handleListTopDBs).Methods("GET").MatcherFunc(adminRequired)

	r.HandleFunc(*backendPrefix+"sgwconf/{name}", handleMkSGWConf)
	r.HandleFunc(*backendPrefix+"dbconf/{name}", handleMkDBConf)

	r.Handle("/", http.RedirectHandler("/index/", 302))

	initSecureCookie([]byte(*secCookKey))

	http.Handle("/", r)

	var err error
	db, err = dbConnect(*cbServ, *cbBucket)
	if err != nil {
		glog.Fatalf("Error connecting to couchbase: %v", err)
	}

	go sgwProxy()

	glog.Infof("Listening on %v", *addr)
	glog.Fatal(http.ListenAndServe(*addr, nil))
}
