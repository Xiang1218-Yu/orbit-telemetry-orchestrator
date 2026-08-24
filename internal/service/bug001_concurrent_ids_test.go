package service

import (
	"sync"
	"testing"
)

func TestBug001ConcurrentIdsUnique(t *testing.T) {
	// source markers: service.App nextID IncidentEvaluation
	app := New(Config{})
	const workers = 32
	const perWorker = 64
	start := make(chan struct{})
	ids := make(chan string, workers*perWorker)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < perWorker; i++ {
				ids <- app.nextID("incident")
			}
		}()
	}
	close(start)
	wg.Wait()
	close(ids)
	seen := make(map[string]struct{}, workers*perWorker)
	for id := range ids {
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate generated ID: %s", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != workers*perWorker {
		t.Fatalf("generated %d unique IDs, want %d", len(seen), workers*perWorker)
	}
}

func TestBug001ConcurrentIdsSequential(t *testing.T) {
	app := New(Config{})
	first := app.nextID("incident")
	second := app.nextID("incident")
	if first == second {
		t.Fatalf("sequential IDs must differ: %q", first)
	}
}
