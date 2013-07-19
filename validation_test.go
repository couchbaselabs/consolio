package main

import (
	"testing"
)

func TestValidDBName(t *testing.T) {
	valid := []string{"a", "great", "db2"}
	invalid := []string{"", "wtf?", "_lodash", "spacious name"}

	for _, d := range valid {
		if !isValidDBName(d) {
			t.Errorf("Expected this to be valid: %q", d)
		}
	}

	for _, d := range invalid {
		if isValidDBName(d) {
			t.Errorf("Expected this to be invalid: %q", d)
		}
	}
}
