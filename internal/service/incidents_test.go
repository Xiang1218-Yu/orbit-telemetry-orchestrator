package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/queue"
	"orbit-telemetry-orchestrator/internal/store"
)

// newTestApp builds an App backed by an in-memory store and a multi-worker
// queue so concurrent incident evaluation mirrors production wiring.
func newTestApp(t *testing.T, workers int) *App {
	t.Helper()
	repository := store.NewMemory()
	workers_ := queue.New(queue.Config{Workers: workers, Buffer: 256})
	app := New(Config{Repository: repository, Queue: workers_})
	workers_.Register("incident-evaluation", app.ProcessIncidentEvaluation)
	workers_.Register("response-action", app.ProcessResponseAction)
	workers_.Register("evidence-rollup", app.ProcessEvidenceRollup)
	workers_.Start()
	t.Cleanup(workers_.Stop)
	return app
}

func makeAnomaly(id, deviceID, signal string, severity int) domain.Anomaly {
	now := time.Now().UTC()
	return domain.Anomaly{
		ID: id, DeviceID: deviceID, Signal: signal, Severity: severity,
		Status: domain.AnomalyOpen, CreatedAt: now, UpdatedAt: now,
	}
}

// TestCorrelateIncidentBurstKeepsAllAnomalies reproduces the reported incident
// aggregation bug: when a burst of anomalies for one device arrives at once,
// they must all attach to a single open incident rather than being split or
// dropped by a lost update.
func TestCorrelateIncidentBurstKeepsAllAnomalies(t *testing.T) {
	const device = "device-burst"
	app := newTestApp(t, 4)
	ctx := context.Background()

	// Persist the anomalies first so correlateIncident can reference them by ID
	// when building the incident (matches how EvaluateTelemetry emits them).
	anomalies := make([]domain.Anomaly, 12)
	for i := range anomalies {
		a := makeAnomaly(fmt.Sprintf("an-%d", i), device, "cpu_temp", 3)
		anomalies[i] = a
		if err := app.config.Repository.PutAnomaly(a); err != nil {
			t.Fatalf("put anomaly %d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	for i := range anomalies {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, ok := app.correlateIncident(ctx, anomalies[i])
			if !ok {
				t.Errorf("anomaly %d not correlated", i)
			}
		}(i)
	}
	wg.Wait()

	incidents := app.config.Repository.ListIncidents(device, "", 0)
	if len(incidents) != 1 {
		t.Fatalf("expected a single incident for the burst, got %d", len(incidents))
	}
	incident := incidents[0]
	if got := len(incident.AnomalyIDs); got != len(anomalies) {
		t.Fatalf("incident lost anomalies: got %d attached, want %d", got, len(anomalies))
	}

	seen := make(map[string]bool, len(anomalies))
	for _, id := range incident.AnomalyIDs {
		seen[id] = true
	}
	for i, a := range anomalies {
		if !seen[a.ID] {
			t.Errorf("anomaly %d (%s) missing from incident", i, a.ID)
		}
	}
}

// TestAppendEvidenceActionConcurrent keeps the open-incident aggregation
// correct when later evidence and response actions are attached concurrently,
// which is the second symptom in the report.
func TestAppendEvidenceActionConcurrent(t *testing.T) {
	const device = "device-evidence"
	app := newTestApp(t, 4)
	ctx := context.Background()

	anomaly := makeAnomaly("an-e1", device, "memory_pressure", 4)
	if err := app.config.Repository.PutAnomaly(anomaly); err != nil {
		t.Fatal(err)
	}
	incident, ok := app.correlateIncident(ctx, anomaly)
	if !ok {
		t.Fatal("baseline anomaly not correlated")
	}

	// Attach evidence concurrently.
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := app.AddEvidence(ctx, "operator", incident.ID, EvidenceInput{
				ID: fmt.Sprintf("ev-%d", i), Kind: domain.EvidenceNote,
				Title: "note", Body: "payload", Source: "probe",
			})
			if err != nil {
				t.Errorf("attach evidence %d: %v", i, err)
			}
		}(i)
	}
	// Attach response actions concurrently.
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := app.CreateAction(ctx, "operator", ActionInput{
				ID: fmt.Sprintf("ac-%d", i), IncidentID: incident.ID,
				Type: domain.ActionNotify, MaxAttempts: 1,
				Idempotency: fmt.Sprintf("idem-%d", i), Reason: "auto",
			})
			if err != nil {
				t.Errorf("create action %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	final, err := app.GetIncident(ctx, incident.ID)
	if err != nil {
		t.Fatal(err)
	}
	// The rollup job may attach an async snapshot evidence, so check by
	// membership rather than exact count: all 8 operator notes must survive.
	evidence := make(map[string]bool, len(final.EvidenceIDs))
	for _, id := range final.EvidenceIDs {
		evidence[id] = true
	}
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("ev-%d", i)
		if !evidence[id] {
			t.Errorf("evidence %s lost from incident (got %d total)", id, len(final.EvidenceIDs))
		}
	}
	// Actions have no async rollup, so the exact count must hold.
	if len(final.ActionIDs) != 8 {
		t.Errorf("actions lost: got %d, want 8", len(final.ActionIDs))
	}
	actions := make(map[string]bool, len(final.ActionIDs))
	for _, id := range final.ActionIDs {
		actions[id] = true
	}
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("ac-%d", i)
		if !actions[id] {
			t.Errorf("action %s lost from incident", id)
		}
	}
}
