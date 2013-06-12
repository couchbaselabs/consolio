package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/couchbaselabs/go-couchbase"
)

type viewMarker struct {
	Version   int       `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

const ddocKey = "/@consolioddocVersion"
const ddocVersion = 0
const designDoc = ``

func updateView(d *couchbase.Bucket) error {
	marker := viewMarker{}
	err := d.Get(ddocKey, &marker)
	if err != nil {
		log.Printf("Error checking view version: %v", err)
	}
	if marker.Version < ddocVersion {
		log.Printf("Installing new version of views (old version=%v)",
			marker.Version)
		doc := json.RawMessage([]byte(designDoc))
		err = d.PutDDoc("cbugg", &doc)
		if err != nil {
			return err
		}
		marker.Version = ddocVersion
		marker.Timestamp = time.Now().UTC()
		marker.Type = "ddocmarker"

		return d.Set(ddocKey, 0, &marker)
	}
	return nil
}

func dbConnect(serv, bucket string) (*couchbase.Bucket, error) {

	log.Printf("Connecting to couchbase bucket %v at %v",
		bucket, serv)
	rv, err := couchbase.GetBucket(serv, "default", bucket)
	if err != nil {
		return nil, err
	}

	if designDoc != "" {
		err = updateView(rv)
	}

	return rv, err
}
