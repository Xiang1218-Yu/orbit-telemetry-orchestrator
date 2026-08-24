package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
)

func TestBug009IncidentCorrelationConcurrentAnomalies(t *testing.T) {
	// source markers: service.correlateIncident store.FindOpenIncident UpdateIncident
	app := New(Config{})
	const callers = 64
	seed := domain.Anomaly{
		ID: "anomaly-seed", DeviceID: "device-1", Signal: "temperature",
		Severity: 3, CreatedAt: time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
	}
	if err := app.config.Repository.PutAnomaly(seed); err != nil {
		t.Fatal(err)
	}
	if _, ok := app.correlateIncident(context.Background(), seed); !ok {
		t.Fatal("seed anomaly did not open an incident")
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			anomaly := domain.Anomaly{
				ID: "anomaly-" + itoa(index), DeviceID: "device-1", Signal: "temperature",
				Severity: 3, CreatedAt: time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
			}
			if err := app.config.Repository.PutAnomaly(anomaly); err != nil {
				t.Errorf("put anomaly %d: %v", index, err)
				return
			}
			_, _ = app.correlateIncident(context.Background(), anomaly)
		}(i)
	}
	close(start)
	wg.Wait()
	incidents := app.config.Repository.ListIncidents("device-1", "", 0)
	if len(incidents) != 1 {
		t.Fatalf("incident count = %d, want 1", len(incidents))
	}
	if got := len(incidents[0].AnomalyIDs); got != callers+1 {
		t.Fatalf("incident anomaly count = %d, want %d", got, callers+1)
	}
}

func TestBug009IncidentCorrelationSingleAnomaly(t *testing.T) {
	app := New(Config{})
	anomaly := domain.Anomaly{
		ID: "anomaly-single", DeviceID: "device-1", Signal: "temperature",
		Severity: 2, CreatedAt: time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
	}
	if err := app.config.Repository.PutAnomaly(anomaly); err != nil {
		t.Fatal(err)
	}
	incident, ok := app.correlateIncident(context.Background(), anomaly)
	if !ok || incident.ID == "" {
		t.Fatalf("single anomaly did not open an incident")
	}
}
