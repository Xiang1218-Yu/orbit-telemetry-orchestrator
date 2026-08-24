package events

import (
	"testing"
	"time"
)

func TestBug004EventPayloadIsolationNestedState(t *testing.T) {
	// source markers: events.Bus Publish Subscription Events
	bus := NewBus(Config{Buffer: 2})
	first := bus.Subscribe("")
	second := bus.Subscribe("")
	defer first.Close()
	defer second.Close()
	event := bus.NewEvent("incident.opened", "inc-1", "system", map[string]any{
		"device": map[string]any{"site": "lab-a"},
	}, nowForBug004())
	bus.Publish(event)
	firstEvent := <-first.Events()
	secondEvent := <-second.Events()
	firstEvent.Payload["device"].(map[string]any)["site"] = "changed"
	got := secondEvent.Payload["device"].(map[string]any)["site"]
	if got != "lab-a" {
		t.Fatalf("subscriber payload was mutated through another subscriber: %v", got)
	}
}

func TestBug004EventPayloadIsolationDelivery(t *testing.T) {
	bus := NewBus(Config{Buffer: 2})
	sub := bus.Subscribe("incident.opened")
	defer sub.Close()
	bus.Publish(bus.NewEvent("incident.opened", "inc-2", "system", nil, nowForBug004()))
	if got := (<-sub.Events()).Subject; got != "inc-2" {
		t.Fatalf("subject = %q, want inc-2", got)
	}
}

func nowForBug004() (value time.Time) {
	return time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
}
