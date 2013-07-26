package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang/glog"

	"github.com/couchbaselabs/consolio/tools"
	"github.com/couchbaselabs/consolio/types"
)

var (
	backendUrl = flag.String("backend", "", "URL to consolio backend API")
)

func updateItem(t, dbname, urlstring string) error {
	u := *backendUrl + "update/" + t + "/" + dbname
	pu, err := url.Parse(urlstring)
	if err == nil {
		pu.User = nil
		urlstring = pu.String()
	}
	glog.Infof("Posting url=%v to %v", urlstring, u)
	res, err := http.PostForm(u, url.Values{"url": []string{urlstring}})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	glog.Infof("Got %v", res.Status)
	if res.StatusCode != 204 {
		return fmt.Errorf("HTTP error marking task done: %v", res.Status)
	}

	return nil
}

func markDone(id, errstr string) error {
	u := *backendUrl + "todo/" + id
	res, err := http.PostForm(u, url.Values{"error": []string{errstr}})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 204 {
		return fmt.Errorf("HTTP error marking task done: %v", res.Status)
	}

	return nil
}

func maybeAppend(errs []error, e error) []error {
	if e != nil {
		errs = append(errs, e)
	}
	return errs
}

func processTodo() error {
	glog.Infof("Processing TODOs...")
	res, err := http.Get(*backendUrl + "todo/")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("Bad HTTP response: %v", res.Status)
	}

	data := []consolio.ChangeEvent{}
	d := json.NewDecoder(res.Body)
	err = d.Decode(&data)
	if err != nil {
		return err
	}

	for _, e := range data {
		pw, err := consoliotools.Decrypt(e.Item.Password)
		if err != nil {
			return err
		}

		errs := []error{}
		for _, h := range handlers {
			errs = maybeAppend(errs, h(e, pw))
		}

		if len(errs) == 0 {
			err = markDone(e.ID, "")
			if err != nil {
				return err
			}
		} else {
			markDone(e.ID, fmt.Sprintf("%v", errs))
			glog.Infof("There were errors processing event: %v", errs)
		}
	}

	return nil
}
