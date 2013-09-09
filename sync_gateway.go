package main

import (
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/gomemcached"
	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"github.com/couchbaselabs/consolio/types"
)

const sgwType = "sync_gateway"

var slumdb = flag.String("slum", "http://localhost:8091/",
	"URL to syncgw's couchbase")
var sgwPersonaOrigin = flag.String("sgw.personaOrigin",
	"http://sync.couchbasecloud.com/",
	"persona origin URL for the sync gateway")
var sgwPersonaRegister = flag.Bool("sgw.personaRegister",
	false, "automatically provision persona authenticated users")

var notAdded = errors.New("Not added")

func generateRandomBucket(owner, genFor string) (*consolio.Item, error) {
	d := consolio.Item{
		Name:     "dbgen-" + randstring(12),
		Password: encrypt(randstring(12)),
		Type:     "database",
		Owner:    owner,
		Enabled:  true,
		URL:      *slumdb,
		LastMod:  time.Now().UTC(),
		ExtraInfo: map[string]interface{}{
			"generated_for": genFor,
		},
	}

	added, err := db.Add("db-"+d.Name, 0, d)
	if err != nil {
		return nil, err
	}
	if !added {
		return nil, notAdded
	}

	return &d, recordEvent("create", d)
}

func handleNewSGW(w http.ResponseWriter, req *http.Request) {
	me := whoami(req)
	d := consolio.Item{
		Name:    strings.TrimSpace(req.FormValue("name")),
		Type:    sgwType,
		Owner:   me.Id,
		Enabled: true,
		LastMod: time.Now().UTC(),
		ExtraInfo: map[string]interface{}{
			"dbname": strings.TrimSpace(req.FormValue("dbname")),
			"sync":   strings.TrimSpace(req.FormValue("syncfun")),
		},
	}

	bname, _ := d.ExtraInfo["dbname"].(string)
	if bname != "" {
		bucket := consolio.Item{}
		err := db.Get("db-"+bname, &bucket)
		if err == nil {
			d.ExtraInfo["db_pass"] = bucket.Password
		} else {
			showError(w, req, "Error validating bucket: "+err.Error(), 500)
			return
		}
		d.ExtraInfo["server"] = bucket.URL
		glog.Infof("Using existing bucket: %v for %v", bucket.Name, d.Name)
	} else {
		bucket, err := generateRandomBucket(me.Id, d.Name)
		if err != nil {
			showError(w, req,
				"Could not setup creation of tmp db: "+err.Error(), 500)
			return
		}
		glog.Infof("Created random bucket: %v for %v", bucket.Name, d.Name)
		d.ExtraInfo["db_pass"] = bucket.Password
		d.ExtraInfo["server"] = bucket.URL
		d.ExtraInfo["generated_db"] = bucket.Name
		d.ExtraInfo["dbname"] = bucket.Name
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

// TODO: func handleUpdateSGWConf

func handleMkSGWConf(w http.ResponseWriter, req *http.Request) {
	name := mux.Vars(req)["name"]
	d := consolio.Item{}
	err := db.Get("sgw-"+name, &d)
	if err != nil {
		glog.Warningf("Error retrieving sync gateway %q: %v", name, err)
		showError(w, req, "Unknown sync gateway", 404)
		return
	}

	if !d.Enabled {
		glog.Warningf("Trying to activate disabled DB %q", name)
		showError(w, req, "Disabled database", 404)
		return
	}

	mustEncode(w, d)
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

	if ob, ok := d.ExtraInfo["generated_db"]; ok {
		glog.Infof("Issuing delete of automatically generated db: %v", ob)
		mux.Vars(req)["name"] = ob.(string)
		handleDeleteDB(w, req)
	} else {
		w.WriteHeader(204)
	}
}

func handleListSGWs(w http.ResponseWriter, req *http.Request) {
	listItem(w, req, sgwType)
}
