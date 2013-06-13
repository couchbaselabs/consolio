package main

import (
	"log"

	"github.com/couchbaselabs/go-couchbase"
	"github.com/couchbaselabs/go-couchbase/util"
)

const ddocKey = "consolio"
const markerKey = "/@consolioddocVersion"
const ddocVersion = 1
const ddocBody = `{
  "id": "_design/consolio",
  "views": {
    "databases": {
      "map": "function (doc, meta) {\n  if (doc.type === 'database') {\n    emit([doc.owner, doc.name], doc.size);\n  }\n}",
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
