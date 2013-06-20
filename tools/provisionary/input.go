package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/couchbaselabs/consolio/types"
)

var (
	todoUrl = flag.String("url", "", "URL to TODO API")
)

func markDone(id string) error {
	u := *todoUrl + id
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
	res, err := http.Get(*todoUrl)
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
		pw, err := decrypt(e.Database.Password)
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
