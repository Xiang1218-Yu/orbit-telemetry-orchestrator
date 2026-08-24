package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/store"
)

var zeroTime time.Time

// newAppWithIncident stands up an App backed by an in-memory repository that
// already contains an open incident the action can be created against.
func newAppWithIncident(t *testing.T) (*App, domain.Incident) {
	t.Helper()
	repository := store.NewMemory()
	incident, err := domain.NewIncident("incident-1", "device-1", "demo incident", 3, "anomaly-1", repository.Counts(zeroTime).At)
	if err != nil {
		t.Fatalf("seed incident: %v", err)
	}
	if err := repository.PutIncident(incident); err != nil {
		t.Fatalf("put incident: %v", err)
	}
	app := New(Config{Repository: repository})
	return app, incident
}

// A retry of the same notification action (same idempotency key) must resolve
// to the same canonical, persisted action — never a freshly generated ID that
// the action list cannot find. This is the operator-facing failure: the API
// returned a new action number that the list could not show, so the operator
// believed the notification was queued when it was not.
func TestCreateAction_IdempotentRetryReturnsSameAction(t *testing.T) {
	app, incident := newAppWithIncident(t)
	ctx := context.Background()
	input := ActionInput{
		ID:          "act-1",
		IncidentID:  incident.ID,
		Type:        domain.ActionNotify,
		MaxAttempts: 3,
		Idempotency: "client-key-1",
		Reason:      "notify on-call",
	}

	first, err := app.CreateAction(ctx, "operator", input)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Retry reuses the idempotency key but, as a buggy client or ops page
	// would, generates a brand-new action ID.
	retryInput := input
	retryInput.ID = "act-2"
	retry, err := app.CreateAction(ctx, "operator", retryInput)
	if err != nil {
		t.Fatalf("retry create: %v", err)
	}

	if retry.ID != first.ID {
		t.Fatalf("retry returned a different action: got %q want %q", retry.ID, first.ID)
	}
	if retry.Idempotency != first.Idempotency {
		t.Fatalf("retry idempotency mismatch: got %q want %q", retry.Idempotency, first.Idempotency)
	}

	actions := app.config.Repository.ListActions(incident.ID, "", 0)
	if len(actions) != 1 {
		t.Fatalf("expected exactly one persisted action, got %d: %+v", len(actions), actions)
	}
	if actions[0].ID != first.ID {
		t.Fatalf("persisted action %q does not match returned action %q", actions[0].ID, first.ID)
	}
	// The retry must not have attached a second action to the incident.
	if len(incident.ID) == 0 {
		t.Fatal("incident id is empty")
	}
	stored, err := app.config.Repository.GetIncident(incident.ID)
	if err != nil {
		t.Fatalf("reload incident: %v", err)
	}
	if len(stored.ActionIDs) != 1 || stored.ActionIDs[0] != first.ID {
		t.Fatalf("incident actions = %v, want [%q]", stored.ActionIDs, first.ID)
	}
}

// A genuinely different request (different idempotency key) must still create
// a new, distinct action — idempotency must not collapse unrelated requests.
func TestCreateAction_DistinctKeysCreateDistinctActions(t *testing.T) {
	app, incident := newAppWithIncident(t)
	ctx := context.Background()

	a1, err := app.CreateAction(ctx, "operator", ActionInput{
		IncidentID: incident.ID, Type: domain.ActionNotify, Idempotency: "key-a",
	})
	if err != nil {
		t.Fatalf("create a: %v", err)
	}
	a2, err := app.CreateAction(ctx, "operator", ActionInput{
		IncidentID: incident.ID, Type: domain.ActionNotify, Idempotency: "key-b",
	})
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	if a1.ID == a2.ID {
		t.Fatalf("distinct keys produced the same action id %q", a1.ID)
	}
	if got := len(app.config.Repository.ListActions(incident.ID, "", 0)); got != 2 {
		t.Fatalf("expected two persisted actions, got %d", got)
	}
}

// When the client omits an action ID, the service assigns one so the request
// still resolves to a valid, addressable action.
func TestCreateAction_GeneratesIDWhenAbsent(t *testing.T) {
	app, incident := newAppWithIncident(t)
	action, err := app.CreateAction(context.Background(), "operator", ActionInput{
		IncidentID: incident.ID, Type: domain.ActionNotify, Idempotency: "key-no-id",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if action.ID == "" {
		t.Fatal("service returned an action with an empty id")
	}
	if _, err := app.config.Repository.GetAction(action.ID); err != nil {
		t.Fatalf("generated action %q not persisted: %v", action.ID, err)
	}
}

// An idempotency-key collision on a different action ID resolves to the stored
// action at the repository layer — this is the unit the service relies on.
func TestPutAction_IdempotentResolution(t *testing.T) {
	repository := store.NewMemory()
	original := mustAction(t, "act-1", "key-1")
	if _, err := repository.PutAction(original); err != nil {
		t.Fatalf("first put: %v", err)
	}

	// Same key, different ID, as a retry would send.
	duplicate := mustAction(t, "act-2", "key-1")
	got, err := repository.PutAction(duplicate)
	if !errors.Is(err, store.ErrExists) {
		t.Fatalf("duplicate put err = %v, want ErrExists", err)
	}
	if got.ID != original.ID {
		t.Fatalf("duplicate put returned %q, want canonical %q", got.ID, original.ID)
	}
	if got.Idempotency != original.Idempotency {
		t.Fatalf("returned action lost idempotency key: got %q want %q", got.Idempotency, original.Idempotency)
	}
}

func mustAction(t *testing.T, id, key string) domain.ResponseAction {
	t.Helper()
	action, err := domain.NewAction(id, "incident-1", "device-1", domain.ActionNotify, 1, key, "reason", zeroTime)
	if err != nil {
		t.Fatalf("new action %s: %v", id, err)
	}
	return action
}
