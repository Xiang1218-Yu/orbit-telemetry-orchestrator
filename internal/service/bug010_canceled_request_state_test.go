package service

import (
	"context"
	"testing"

	"orbit-telemetry-orchestrator/internal/store"
)

func TestBug010CanceledRequestStateNoPersistence(t *testing.T) {
	// source markers: service.CreateDevice store.Memory POST /v1/devices context.Context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app := New(Config{})
	_, err := app.CreateDevice(ctx, "operator", DeviceInput{
		ID: "device-canceled", Name: "Canceled Gateway", Site: "lab-a", Model: "orbit-gw",
	})
	if err != context.Canceled {
		t.Fatalf("CreateDevice error = %v, want context canceled", err)
	}
	if _, err := app.config.Repository.GetDevice("device-canceled"); err != store.ErrNotFound {
		t.Fatalf("canceled request persisted device, lookup error = %v", err)
	}
}

func TestBug010CanceledRequestStateActiveCreation(t *testing.T) {
	app := New(Config{})
	device, err := app.CreateDevice(context.Background(), "operator", DeviceInput{
		ID: "device-active", Name: "Active Gateway", Site: "lab-a", Model: "orbit-gw",
	})
	if err != nil {
		t.Fatal(err)
	}
	if device.ID != "device-active" {
		t.Fatalf("device ID = %q, want device-active", device.ID)
	}
}
