package queue

import (
	"context"
	"testing"
	"time"
)

func TestBug002WorkerContextPropagation(t *testing.T) {
	// source markers: service.App ProcessIncidentEvaluation EvaluateTelemetry
	q := New(Config{Workers: 1, Buffer: 1})
	started := make(chan struct{})
	observed := make(chan struct{})
	q.Register("incident-evaluation", func(ctx context.Context, _ Job) error {
		close(started)
		select {
		case <-ctx.Done():
			close(observed)
		case <-time.After(200 * time.Millisecond):
		}
		return nil
	})
	q.Start()
	if _, err := q.Submit("incident-evaluation", struct{}{}); err != nil {
		t.Fatal(err)
	}
	<-started
	q.Stop()
	select {
	case <-observed:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("worker did not receive queue cancellation")
	}
}

func TestBug002WorkerContextHandler(t *testing.T) {
	q := New(Config{Workers: 1, Buffer: 1})
	done := make(chan struct{})
	q.Register("incident-evaluation", func(context.Context, Job) error {
		close(done)
		return nil
	})
	q.Start()
	if _, err := q.Submit("incident-evaluation", struct{}{}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("registered handler did not run")
	}
	q.Stop()
}
