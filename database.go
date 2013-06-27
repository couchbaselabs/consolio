package main

import (
	"log"

	"github.com/couchbaselabs/go-couchbase"
	"github.com/couchbaselabs/go-couchbase/util"
)

const ddocKey = "consolio"
const markerKey = "/@consolioddocVersion"
const ddocVersion = 4
const ddocBody = `{
  "id": "_design/consolio",
  "views": {
    "webhooks": {
      "map": "function (doc, meta) {\n  if (doc.type === 'webhook') {\n    emit(doc.name, doc.url);\n  }\n}"
    },
    "events": {
      "map": "function (doc, meta) {\n  if (doc.type === 'create' || doc.type === 'delete') {\n    emit([doc.processed ? 'done' : 'todo', doc.ts], null);\n  }\n}"
    },
    "items": {
      "map": "function (doc, meta) {\n  if (doc.owner && doc.type && doc.name) {\n    emit([doc.type, doc.owner, doc.name], doc.size || 0);\n  }\n}",
      "reduce": "_sum"
    }
  }
}`

func dbConnect(serv, bucket string) (*couchbase.Bucket, error) {

	log.Printf("Connecting to couchbase bucket %v at %v",
		bucket, serv)
	rv, err := couchbase.GetBucket(serv, "default", bucket)
	if err != nil {
		return nil, err
	}

	return rv, couchbaseutil.UpdateView(rv,
		ddocKey, markerKey, ddocBody, ddocVersion)
}
