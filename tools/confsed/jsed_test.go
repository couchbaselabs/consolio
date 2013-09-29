package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

var sample = []byte("[1,2,\"three \\\"thirty\", 4, \"four\", {\"what\":\"yeah\"}]")

func verify(t *testing.T, trans transformer, in, exp []byte) {
	got, err := ioutil.ReadAll(rewriteJson(bytes.NewReader(in), trans))
	if err != nil {
		t.Errorf("Error reading stuff: %v", err)
	}
	if !bytes.Equal(exp, got) {
		t.Errorf("Expected\n%s\ngot\n%s", exp, got)
	}
}

func TestIdentity(t *testing.T) {
	verify(t, identity, sample, sample)
}

func TestStripQuote(t *testing.T) {
	exp := []byte("[1,2,\"three thirty\", 4, \"four\", {\"what\":\"yeah\"}]")
	verify(t, func(s string) string {
		return strings.Replace(s, `"`, "", -1)
	}, sample, exp)
}
