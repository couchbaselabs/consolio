package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/gomemcached"
	"github.com/gorilla/mux"

	"github.com/couchbaselabs/consolio/types"
)

const sgwType = "sync_gateway"

var slumdb = flag.String("slum", "http://localhost:8091/",
	"URL to syncgw's couchbase")

func handleNewSGW(w http.ResponseWriter, req *http.Request) {
	d := consolio.Item{
		Name:    strings.TrimSpace(req.FormValue("name")),
		Type:    sgwType,
		Owner:   whoami(req).Id,
		Enabled: true,
		LastMod: time.Now().UTC(),
		ExtraInfo: map[string]interface{}{
			"dbname": strings.TrimSpace(req.FormValue("dbname")),
			"sync":   strings.TrimSpace(req.FormValue("syncfun")),
		},
	}

	bname, ok := d.ExtraInfo["dbname"].(string)
	if ok {
		bucket := consolio.Item{}
		err := db.Get("db-"+bname, &bucket)
		if err == nil {
			d.ExtraInfo["db_pass"] = bucket.Password
		}
	}

	if b, _ := strconv.ParseBool(req.FormValue("guest")); b {
		guestInfo := json.RawMessage(`{"disabled": false, "admin_channels": ["*"] }`)
		d.ExtraInfo["users"] = map[string]interface{}{"GUEST": &guestInfo}
	}

	if !isValidDBName(d.Name) {
		showError(w, req, "Invalid DB Name", 400)
		return
	}

	added, err := db.Add("sgw-"+d.Name, 0, d)
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

func handleMkSGWConf(w http.ResponseWriter, req *http.Request) {
	viewRes := struct {
		Rows []struct {
			Key []string
			Doc struct {
				Json consolio.Item
			}
		}
	}{}

	empty := &json.RawMessage{'{', '}'}
	err := db.ViewCustom("consolio", "items",
		map[string]interface{}{
			"reduce":       false,
			"include_docs": true,
			"stale":        false,
			"start_key":    []interface{}{sgwType},
			"end_key":      []interface{}{sgwType, empty},
		},
		&viewRes)
	if err != nil {
		showError(w, req, "Did Error listing stuff: "+
			err.Error(), 500)
		return
	}

	rv := struct {
		Intf      string                            `json:"interface"`
		AdminIntf string                            `json:"adminInterface"`
		Log       []string                          `json:"log"`
		Databases map[string]map[string]interface{} `json:"databases"`
	}{
		Intf:      ":4984",
		AdminIntf: ":4985",
		Log:       []string{"REST"},
		Databases: map[string]map[string]interface{}{},
	}

	for _, r := range viewRes.Rows {
		h := r.Doc.Json.ExtraInfo
		h["server"] = *slumdb
		h["bucket"] = h["dbname"]

		delete(h, "dbname")
		rv.Databases[r.Key[2]] = h
	}

	mustEncode(w, rv)
}

func handleGetSGW(w http.ResponseWriter, req *http.Request) {
	d := consolio.Item{}
	err := db.Get("sgw-"+mux.Vars(req)["name"], &d)
	switch {
	case gomemcached.IsNotFound(err):
		showError(w, req, "Not found", 404)
	case err != nil:
		showError(w, req, err.Error(), 500)
	case d.Type != sgwType:
		showError(w, req, "Incorrect type", 400)
	case d.Owner != whoami(req).Id:
		showError(w, req, "Not your SGW", 403)
	default:
		mustEncode(w, d)
	}
}

func handleDeleteSGW(w http.ResponseWriter, req *http.Request) {
	d := consolio.Item{}
	k := "sgw-" + mux.Vars(req)["name"]
	err := db.Get(k, &d)
	switch {
	case gomemcached.IsNotFound(err):
		showError(w, req, "Not found", 404)
	case err != nil:
		showError(w, req, err.Error(), 500)
	case d.Type != sgwType:
		showError(w, req, "Incorrect type", 400)
	case d.Owner != whoami(req).Id:
		showError(w, req, "Not your SGW", 403)
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

func handleListSGWs(w http.ResponseWriter, req *http.Request) {
	listItem(w, req, sgwType)
}
