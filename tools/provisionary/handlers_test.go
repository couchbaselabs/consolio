package main

import (
	"bytes"
	"testing"
)

var testObscure = `{"bucket":"dbgen-wHXzK8F9Llvz","generated_db":"dbgen-wHXzK8F9Llvz","server":"http://dbgen-wHXzK8F9Llvz:ZXqcF5ARZOKd@db1.couchbasecloud.com:8091/","sync":"function(doc) {\n  channel(doc.channels);\n}","users":{"GUEST":{"admin_channels":["*"],"disabled":false}}}`
var testObscured = `{"bucket":"dbgen-wHXzK8F9Llvz","generated_db":"dbgen-wHXzK8F9Llvz","server":"http://dbgen-wHXzK8F9Llvz:xxxxxxxxxxxx@db1.couchbasecloud.com:8091/","sync":"function(doc) {\n  channel(doc.channels);\n}","users":{"GUEST":{"admin_channels":["*"],"disabled":false}}}`

func TestObscurePassword(t *testing.T) {
	got := obscurePassword([]byte(testObscure))
	exp := []byte(testObscured)
	if !bytes.Equal(got, exp) {
		t.Errorf("Expected %s, got %s", exp, got)
	}
}
