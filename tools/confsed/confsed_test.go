package main

import (
	"strings"
	"sync"
	"testing"
)

// Curious how long it takes to make a replacer.  If I decide I can't
// reuse them, I'll know the cost.
func BenchmarkCreateReplacer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		strings.NewReplacer("one", "1", "two", "2")
	}
}

// I assume it's safe to use a replacer concurrently.  This test helps
// me know that it's OK (especially when run with the race detector).
func TestConcurrentReplace(t *testing.T) {
	r := strings.NewReplacer("one", "1", "two", "2")

	pistol := make(chan bool)
	failed := make(chan string)
	wg := &sync.WaitGroup{}
	startConcurrentTest := func() {
		defer wg.Done()
		<-pistol
		for i := 0; i < 100; i++ {
			got := r.Replace("one two three")
			if got != "1 2 three" {
				failed <- got
			}
		}
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go startConcurrentTest()
	}
	close(pistol)

	go func() {
		wg.Wait()
		close(failed)
	}()

	errors := []string{}
	for e := range failed {
		errors = append(errors, e)
	}

	if len(errors) > 0 {
		t.Errorf("Failed to dtrt: %v", errors)
	}
}
