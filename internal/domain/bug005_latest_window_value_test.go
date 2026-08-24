package domain

import (
	"testing"
	"time"
)

func TestBug005LatestWindowValueNewest(t *testing.T) {
	// source markers: service.IngestTelemetry store.Memory BuildWindow
	start := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	samples := []Sample{
		{StreamID: "stream-1", DeviceID: "device-1", Signal: "temperature", Value: 99, ObservedAt: start.Add(2 * time.Minute), Quality: "good"},
		{StreamID: "stream-1", DeviceID: "device-1", Signal: "temperature", Value: 10, ObservedAt: start.Add(1 * time.Minute), Quality: "good"},
	}
	window, err := BuildWindow(samples, start, start.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if window.LastValue != 99 {
		t.Fatalf("latest window value = %v, want 99", window.LastValue)
	}
}

func TestBug005LatestWindowValueCount(t *testing.T) {
	start := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	samples := []Sample{
		{StreamID: "stream-1", DeviceID: "device-1", Signal: "temperature", Value: 10, ObservedAt: start.Add(time.Minute), Quality: "good"},
		{StreamID: "stream-1", DeviceID: "device-1", Signal: "temperature", Value: 99, ObservedAt: start.Add(2 * time.Minute), Quality: "good"},
	}
	window, err := BuildWindow(samples, start, start.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if window.Count != 2 {
		t.Fatalf("window count = %d, want 2", window.Count)
	}
}
