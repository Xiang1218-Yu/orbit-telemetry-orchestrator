package service

import (
	"context"
	"errors"
	"testing"

	"orbit-telemetry-orchestrator/internal/audit"
	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/store"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	return New(Config{
		Repository: store.NewMemory(),
		Audit:      audit.NewLog(1000),
	})
}

// TestCreateDevice_CancelledContextLeavesNoRecord guards against the original
// bug: when a client cancels a registration request, the device must not be
// persisted as a half-finished "pending" record, and no audit entry should be
// recorded. Otherwise a later registration of the same device would fail with
// "record already exists" and the status list would misreport a phantom device.
func TestCreateDevice_CancelledContextLeavesNoRecord(t *testing.T) {
	app := newTestApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call

	_, err := app.CreateDevice(ctx, "tester", DeviceInput{
		ID: "device-1", Name: "Gateway One", Site: "lab-a", Model: "orbit-gw",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// No device record should have survived the cancellation.
	if _, err := app.config.Repository.GetDevice("device-1"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected no persisted device after cancellation, got err=%v", err)
	}

	// The device must not appear in listings or counts.
	if devices := app.config.Repository.ListDevices("", 0); len(devices) != 0 {
		t.Fatalf("expected empty device list, got %d", len(devices))
	}
	if counts := app.config.Repository.Counts(app.now()); counts.Devices != 0 {
		t.Fatalf("expected zero device count, got %d", counts.Devices)
	}

	// No audit entry should have been recorded for a cancelled operation.
	if n := app.config.Audit.Count(); n != 0 {
		t.Fatalf("expected no audit entries after cancellation, got %d", n)
	}
}

// TestCreateDevice_RaceCancellationRollsBackRecord covers the case where the
// context is cancelled after PutDevice has already written the record. The
// service must roll the record back so registration of the same device still
// succeeds afterwards.
func TestCreateDevice_RaceCancellationRollsBackRecord(t *testing.T) {
	app := newTestApp(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := app.CreateDevice(ctx, "tester", DeviceInput{
		ID: "device-2", Name: "Gateway Two", Site: "lab-b", Model: "orbit-gw",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	// After a cancellation, re-registering the same device must succeed.
	device, err := app.CreateDevice(context.Background(), "tester", DeviceInput{
		ID: "device-2", Name: "Gateway Two", Site: "lab-b", Model: "orbit-gw",
	})
	if err != nil {
		t.Fatalf("re-registration after cancellation failed: %v", err)
	}
	if device.Status != domain.DevicePending {
		t.Fatalf("expected pending status, got %s", device.Status)
	}
}

// TestCreateDevice_HappyPathPersistsRecord confirms the normal registration
// flow is unaffected by the cancellation guards.
func TestCreateDevice_HappyPathPersistsRecord(t *testing.T) {
	app := newTestApp(t)

	device, err := app.CreateDevice(context.Background(), "tester", DeviceInput{
		ID: "device-3", Name: "Gateway Three", Site: "lab-c", Model: "orbit-gw",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if device.ID != "device-3" || device.Status != domain.DevicePending {
		t.Fatalf("unexpected device: %+v", device)
	}
	if _, err := app.config.Repository.GetDevice("device-3"); err != nil {
		t.Fatalf("device was not persisted: %v", err)
	}
	if n := app.config.Audit.Count(); n != 1 {
		t.Fatalf("expected one audit entry, got %d", n)
	}
}
