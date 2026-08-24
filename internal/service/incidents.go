package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"orbit-telemetry-orchestrator/internal/aggregate"
	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/store"
)

// correlateIncident attaches anomaly to the device's open incident, opening a
// new one only when none exists. The whole find-or-append / find-or-create
// sequence runs atomically inside the store (UpsertIncidentAnomaly), so a burst
// of anomalies for the same device lands on a single incident: the first worker
// creates it and the rest append to it, rather than each racing to open a
// duplicate that swallows the others' anomalies.
func (a *App) correlateIncident(ctx context.Context, anomaly domain.Anomaly) (domain.Incident, bool) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Incident{}, false
	}
	window := a.now().Add(-10 * time.Minute)
	created := false
	incident, ok, err := a.config.Repository.UpsertIncidentAnomaly(
		anomaly.DeviceID, anomaly.Signal, window, anomaly.ID, anomaly.Severity, a.now(),
		func() (domain.Incident, error) {
			created = true
			return domain.NewIncident(
				a.nextID("incident"), anomaly.DeviceID,
				"Telemetry anomaly: "+anomaly.Signal, anomaly.Severity, anomaly.ID, a.now(),
			)
		},
	)
	if err != nil || !ok {
		return domain.Incident{}, false
	}
	if created {
		a.record("system", "open", "incident", incident.ID, "incident opened from anomaly", nil)
		a.emit("incident.opened", incident.ID, "system", map[string]any{"device_id": incident.DeviceID, "severity": incident.Severity})
		a.submit("evidence-rollup", EvidenceRollupJob{IncidentID: incident.ID})
	}
	return incident, true
}

func (a *App) ListAnomalies(ctx context.Context, deviceID, status string, limit int) ([]domain.Anomaly, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	return a.config.Repository.ListAnomalies(deviceID, status, limit), nil
}

func (a *App) GetIncident(ctx context.Context, id string) (domain.Incident, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Incident{}, err
	}
	return a.config.Repository.GetIncident(id)
}

func (a *App) ListIncidents(ctx context.Context, deviceID, status string, limit int) ([]domain.Incident, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	return a.config.Repository.ListIncidents(deviceID, status, limit), nil
}

func (a *App) TransitionIncident(ctx context.Context, actor, id string, status domain.IncidentStatus) (domain.Incident, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Incident{}, err
	}
	final, err := a.updateIncident(ctx, id, func(incident domain.Incident) (domain.Incident, error) {
		if status == domain.IncidentClosed && !incident.ReadyToClose() {
			return domain.Incident{}, errors.New("incident needs resolved status and evidence before closing")
		}
		if err := incident.Transition(status, a.now()); err != nil {
			return domain.Incident{}, err
		}
		return incident, nil
	})
	if err != nil {
		return domain.Incident{}, err
	}
	a.record(actor, "transition", "incident", id, "incident status changed", map[string]string{"status": string(status)})
	a.emit("incident.status_changed", id, actor, map[string]any{"status": status})
	return final, nil
}

// updateIncident performs a read-modify-write on an incident with optimistic
// concurrency control. The mutate callback receives the latest copy and
// returns the modified value; the write is committed with the version the
// callback saw, and retried from a fresh read when a concurrent writer changed
// the version underneath it (store.ErrConflict). This keeps simultaneous
// writes — anomalies, evidence, actions, status transitions — from silently
// overwriting one another. Mutation must be idempotent since it may run more
// than once. Retry is bounded so a permanently contended record still returns
// instead of looping forever.
func (a *App) updateIncident(ctx context.Context, id string, mutate func(domain.Incident) (domain.Incident, error)) (domain.Incident, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Incident{}, err
	}
	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := a.ensureContext(ctx); err != nil {
			return domain.Incident{}, err
		}
		incident, err := a.config.Repository.GetIncident(id)
		if err != nil {
			return domain.Incident{}, err
		}
		version := incident.Version
		updated, err := mutate(incident)
		if err != nil {
			return domain.Incident{}, err
		}
		if err := a.config.Repository.UpdateIncident(updated, version); err != nil {
			if errors.Is(err, store.ErrConflict) {
				continue
			}
			return domain.Incident{}, err
		}
		return updated, nil
	}
	return domain.Incident{}, errors.New("incident update conflict: too many retries")
}

func (a *App) EvaluateIncident(ctx context.Context, id string) (domain.Incident, error) {
	incident, err := a.GetIncident(ctx, id)
	if err != nil {
		return domain.Incident{}, err
	}
	if incident.Status == domain.IncidentOpen {
		return a.TransitionIncident(ctx, "system", id, domain.IncidentInvestigating)
	}
	return incident, nil
}

func (a *App) SearchIncidents(ctx context.Context, query string, limit int) ([]domain.Incident, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	incidents := a.config.Repository.ListIncidents("", "", 0)
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return incidents, nil
	}
	result := make([]domain.Incident, 0)
	for _, incident := range incidents {
		if strings.Contains(strings.ToLower(incident.Title), query) || strings.Contains(incident.DeviceID, query) {
			result = append(result, incident)
		}
	}
	aggregate.SortAnomalies(nil)
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
