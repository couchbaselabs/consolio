package main

import (
	"github.com/couchbaselabs/go-couchbase"
	"github.com/couchbaselabs/go-couchbase/util"
	"github.com/golang/glog"
)

const ddocKey = "consolio"
const markerKey = "/@consolioddocVersion"
const ddocVersion = 8
const ddocBody = `{
  "id": "_design/consolio",
  "views": {
    "items": {
      "map": "function (doc, meta) {\n  if (doc.owner && doc.type && doc.name) {\n    emit([doc.type, doc.owner, doc.name], doc.size || 0);\n  }\n}",
      "reduce": "_sum"
    },
    "bysize": {
      "map": "function (doc, meta) {\n  if (doc.stats && doc.stats.fileSize) {\n    var name = doc.extra.generated_for || doc.name;\n    emit(doc.stats.fileSize, {name: name, owner: doc.owner});\n  }\n}"
    }
  }
}`

func dbConnect(serv, bucket string) (*couchbase.Bucket, error) {

	glog.Infof("Connecting to couchbase bucket %v at %v",
		bucket, serv)
	rv, err := couchbase.GetBucket(serv, "default", bucket)
	if err != nil {
		return nil, err
	}

	return rv, couchbaseutil.UpdateView(rv,
		ddocKey, markerKey, ddocBody, ddocVersion)
}
