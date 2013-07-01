package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/couchbaselabs/consolio/types"
)

var (
	backendUrl = flag.String("backend", "", "URL to consolio backend API")
)

func updateItem(t, dbname, urlstring string) error {
	u := *backendUrl + "update/" + t + "/" + dbname
	log.Printf("Posting url=%v to %v", urlstring, u)
	res, err := http.PostForm(u, url.Values{"url": []string{urlstring}})
	if err != nil {
		return err
	}
	defer res.Body.Close()
	log.Printf("Got %v", res.Status)
	if res.StatusCode != 204 {
		return fmt.Errorf("HTTP error marking task done: %v", res.Status)
	}

	return nil
}

func markDone(id string) error {
	u := *backendUrl + "todo/" + id
	res, err := http.Post(u, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 204 {
		return fmt.Errorf("HTTP error marking task done: %v", res.Status)
	}

	return nil
}

func processTodo() error {
	log.Printf("Processing TODOs...")
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
		pw, err := decrypt(e.Item.Password)
		if err != nil {
			return err
		}

		for _, h := range handlers {
			h(e, pw)
		}

		err = markDone(e.ID)
		if err != nil {
			return err
		}
	}

	return nil
}
