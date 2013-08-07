package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"

	"github.com/couchbaselabs/consolio/tools"
	"github.com/couchbaselabs/consolio/types"
)

type handler func(consolio.ChangeEvent, string) error

var (
	cbgbUrlFlag     = flag.String("cbgb", "", "CBGB base URL")
	sgwUrlFlag      = flag.String("sgw", "", "URL to sync gateway")
	sgwAdminUrlFlag = flag.String("sgwadmin", "", "URL to sync gateway admin")

	cbgbUrl        string
	cbgbDB         string
	sgwDB          string
	sgwAdmin       string
	handlers       []handler
	cancelRedirect = fmt.Errorf("redirected")
)

func mustParseURL(ustr string) *url.URL {
	u, err := url.Parse(ustr)
	if err != nil {
		glog.Fatalf("Error parsing URL %q: %v", ustr, err)
	}
	return u
}

func initHandlers() {
	handlers = append(handlers, logHandler)

	if *cbgbUrlFlag != "" {
		u := mustParseURL(*cbgbUrlFlag)
		u.Path = "/_api/buckets"
		cbgbUrl = u.String()
		u.Path = "/"
		cbgbDB = u.String()

		handlers = append(handlers, cbgbHandler)
	}

	if *sgwUrlFlag != "" && *sgwAdminUrlFlag != "" {
		u := mustParseURL(*sgwUrlFlag)
		u.Path = "/"
		sgwDB = u.String()

		u = mustParseURL(*sgwAdminUrlFlag)
		u.Path = "/"
		sgwAdmin = u.String()

		handlers = append(handlers, sgwHandler)
	}
}

func logHandler(e consolio.ChangeEvent, pw string) error {
	glog.Infof("Found %v -> %v %v - %q",
		e.ID, e.Type, e.Item.Name, pw)
	return nil
}

func isRedirected(e error) bool {
	if x, ok := e.(*url.Error); ok {
		return x.Err == cancelRedirect
	}
	return false
}

func cbgbHandler(e consolio.ChangeEvent, pw string) error {
	if e.Item.Type != "database" {
		glog.Infof("Ignoring non-database type: %v (%v)",
			e.Item.Name, e.Item.Type)
		return nil
	}
	switch e.Type {
	case "create":
		return cbgbCreate(e.Item.Name, pw)
	case "delete":
		return cbgbDelete(e.Item.Name)
	}
	return fmt.Errorf("Unhandled event type: %v", e.Type)
}

func cbgbDelete(dbname string) error {
	u := cbgbUrl + "/" + dbname
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		glog.Infof("Missing while deleting DB %q, must already be gone", dbname)
		return nil
	}
	if res.StatusCode != 204 {
		return fmt.Errorf("Unexpected HTTP status from cbgb for DELETE %q: %v",
			dbname, res.Status)
	}

	return nil
}

func cbgbCreate(dbname, pw string) error {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return cancelRedirect
		},
	}

	vals := url.Values{}
	vals.Set("name", dbname)
	vals.Set("password", pw)
	vals.Set("quotaBytes", fmt.Sprintf("%d", 256*1024*1024))
	vals.Set("memoryOnly", "0")
	req, err := http.NewRequest("POST", cbgbUrl,
		strings.NewReader(vals.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if !isRedirected(err) {
		if err != nil {
			return err
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 303 {
		bodyText, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP error creating bucket: %v\n%s",
			resp.Status, bodyText)
	}

	return updateItem("db", dbname, cbgbDB+dbname)
}

func sgwHandler(e consolio.ChangeEvent, pw string) error {
	if e.Item.Type != "sync_gateway" {
		glog.Infof("Ignoring non-sgw type: %v (%v)",
			e.Item.Name, e.Item.Type)
		return nil
	}
	switch e.Type {
	case "create":
		return sgwCreate(e, pw)
	case "delete":
		return sgwDelete(e, pw)
	}
	return fmt.Errorf("Unhandled sgw event type: %v", e.Type)
}

func getServerUrl(m map[string]interface{}) string {
	server, ok := m["server"].(string)
	if !ok {
		return server
	}

	bucket, ok := m["bucket"].(string)
	if !ok {
		return server
	}

	pass, ok := m["db_pass"].(string)
	if !ok {
		return server
	}

	u, err := url.Parse(server)
	if err == nil {
		pass, err := consoliotools.Decrypt(pass)
		if err == nil {
			u.User = url.UserPassword(bucket, pass)
		} else {
			glog.Infof("Error decrypting password: %v", err)
		}
		server = u.String()
	}

	return server
}

func sgwCreate(e consolio.ChangeEvent, pw string) error {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return cancelRedirect
		},
	}

	glog.Infof("Got sgw create message: %+v/%+v", e, e.Item)

	conf := map[string]interface{}{}
	for k, v := range e.Item.ExtraInfo {
		conf[k] = v
	}
	conf["bucket"] = conf["dbname"]
	conf["server"] = getServerUrl(conf)
	delete(conf, "dbname")
	delete(conf, "db_pass")

	b, err := json.Marshal(conf)
	if err != nil {
		return err
	}

	glog.Infof("Provisioning with %s", b)

	req, err := http.NewRequest("PUT", sgwAdmin+e.Item.Name+"/",
		bytes.NewReader(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if !isRedirected(err) {
		if err != nil {
			return err
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode == 412 {
		glog.Infof("%q seems to already exist", e.Item.Name)
		return nil
	}
	if resp.StatusCode != 201 {
		bodyText, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP error creating bucket: %v\n%s",
			resp.Status, bodyText)
	}

	return updateItem("sgw", e.Item.Name, sgwDB+e.Item.Name)
}

func sgwDelete(e consolio.ChangeEvent, pw string) error {
	u := sgwAdmin + e.Item.Name + "/"
	req, err := http.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		glog.Infof("Didn't find DB.  Must already be gone.")
		return nil
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Unexpected HTTP status from sgw: %v",
			res.Status)
	}

	return updateItem("sgw", e.Item.Name, sgwDB+e.Item.Name)
}
