package service

import (
	"context"
	"testing"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
)

func TestBug006ActionIdempotencyRepeatedRequest(t *testing.T) {
	// source markers: service.CreateAction store.Memory POST /v1/incidents/{id}/actions
	app := New(Config{})
	incident, err := domain.NewIncident("incident-1", "device-1", "Telemetry anomaly", 3, "anomaly-1", nowForBug006())
	if err != nil {
		t.Fatal(err)
	}
	if err := app.config.Repository.PutIncident(incident); err != nil {
		t.Fatal(err)
	}
	first, err := app.CreateAction(context.Background(), "operator", ActionInput{
		ID: "action-1", IncidentID: incident.ID, Type: domain.ActionNotify,
		MaxAttempts: 1, Idempotency: "same-key", Reason: "notify operator",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.CreateAction(context.Background(), "operator", ActionInput{
		ID: "action-2", IncidentID: incident.ID, Type: domain.ActionNotify,
		MaxAttempts: 1, Idempotency: "same-key", Reason: "retry request",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("repeated request returned %q, want durable action %q", second.ID, first.ID)
	}
	if got := len(app.config.Repository.ListActions(incident.ID, "", 0)); got != 1 {
		t.Fatalf("durable action count = %d, want 1", got)
	}
}

func TestBug006ActionIdempotencyDistinctKeys(t *testing.T) {
	app := New(Config{})
	incident, err := domain.NewIncident("incident-2", "device-1", "Telemetry anomaly", 3, "anomaly-2", nowForBug006())
	if err != nil {
		t.Fatal(err)
	}
	_ = app.config.Repository.PutIncident(incident)
	for i, key := range []string{"key-a", "key-b"} {
		if _, err := app.CreateAction(context.Background(), "operator", ActionInput{
			ID: "action-" + string(rune('a'+i)), IncidentID: incident.ID,
			Type: domain.ActionNotify, MaxAttempts: 1, Idempotency: key,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(app.config.Repository.ListActions(incident.ID, "", 0)); got != 2 {
		t.Fatalf("durable action count = %d, want 2", got)
	}
}

func nowForBug006() time.Time {
	return time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
}
