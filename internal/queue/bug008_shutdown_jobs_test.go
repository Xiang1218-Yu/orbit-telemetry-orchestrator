package queue

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestBug008ShutdownJobsAcceptedWork(t *testing.T) {
	// source markers: queue.Queue Stop Scheduler Drain service.App submit
	q := New(Config{Workers: 1, Buffer: 8})
	var handled atomic.Int32
	q.Register("evidence-rollup", func(context.Context, Job) error {
		handled.Add(1)
		return nil
	})
	q.Start()
	for i := 0; i < 4; i++ {
		if _, err := q.Submit("evidence-rollup", i); err != nil {
			t.Fatal(err)
		}
	}
	q.Stop()
	if got := handled.Load(); got != 4 {
		t.Fatalf("handled accepted jobs = %d, want 4", got)
	}
}

func TestBug008ShutdownJobsEmptyDrain(t *testing.T) {
	q := New(Config{Workers: 1, Buffer: 1})
	scheduler := NewScheduler(q)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := scheduler.Drain(ctx); err != nil {
		t.Fatal(err)
	}
}
