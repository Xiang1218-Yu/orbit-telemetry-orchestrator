package service

import (
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestNextID_ConcurrentUnique reproduces the race that produced duplicate event
// numbers under parallel telemetry evaluation. The previous implementation did a
// non-atomic read-modify-write of App.sequence with a runtime.Gosched() in the
// middle, so concurrent callers observed the same value and returned identical
// IDs — leaving one of the two store records shadowed and audit entries
// mis-attributed. The fix increments the counter atomically.
//
// Run with -race to turn any residual data race into a hard failure.
func TestNextID_ConcurrentUnique(t *testing.T) {
	app := New(Config{})
	const goroutines = 64
	const perGoroutine = 256

	seen := make(chan string, goroutines*perGoroutine)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				seen <- app.nextID("anomaly")
			}
		}()
	}
	wg.Wait()
	close(seen)

	ids := make(map[string]struct{}, goroutines*perGoroutine)
	for id := range seen {
		if _, dup := ids[id]; dup {
			t.Fatalf("duplicate id from nextID: %s", id)
		}
		ids[id] = struct{}{}
	}

	if want := goroutines * perGoroutine; len(ids) != want {
		t.Fatalf("id count = %d, want %d", len(ids), want)
	}
}

// TestNextID_Monotonic ensures the numeric suffix strictly increases across
// sequential calls, so ordering assumptions in the audit trail and store hold.
func TestNextID_Monotonic(t *testing.T) {
	app := New(Config{})
	prev := uint64(0)
	for i := 0; i < 1000; i++ {
		id := app.nextID("incident")
		n, err := strconv.ParseUint(strings.TrimPrefix(id, "incident-"), 10, 64)
		if err != nil {
			t.Fatalf("unexpected id format %q: %v", id, err)
		}
		if n <= prev {
			t.Fatalf("id suffix %d not greater than prev %d at i=%d", n, prev, i)
		}
		prev = n
	}
}
