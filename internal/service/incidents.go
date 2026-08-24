package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"orbit-telemetry-orchestrator/internal/aggregate"
	"orbit-telemetry-orchestrator/internal/domain"
)

func (a *App) correlateIncident(ctx context.Context, anomaly domain.Anomaly) (domain.Incident, bool) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Incident{}, false
	}
	existing, ok := a.config.Repository.FindOpenIncident(anomaly.DeviceID, anomaly.Signal, a.now().Add(-10*time.Minute))
	if ok {
		_ = existing.AddAnomaly(anomaly.ID, anomaly.Severity, a.now())
		_ = a.config.Repository.UpdateIncident(existing)
		return existing, true
	}
	incidentID := a.nextID("incident")
	incident, err := domain.NewIncident(
		incidentID, anomaly.DeviceID,
		"Telemetry anomaly: "+anomaly.Signal, anomaly.Severity, anomaly.ID, a.now(),
	)
	if err != nil {
		return domain.Incident{}, false
	}
	if err := a.config.Repository.PutIncident(incident); err != nil {
		return domain.Incident{}, false
	}
	a.record("system", "open", "incident", incident.ID, "incident opened from anomaly", nil)
	a.emit("incident.opened", incident.ID, "system", map[string]any{"device_id": incident.DeviceID, "severity": incident.Severity})
	a.submit("evidence-rollup", EvidenceRollupJob{IncidentID: incident.ID})
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
	incident, err := a.config.Repository.GetIncident(id)
	if err != nil {
		return domain.Incident{}, err
	}
	if status == domain.IncidentClosed && !incident.ReadyToClose() {
		return domain.Incident{}, errors.New("incident needs resolved status and evidence before closing")
	}
	if err := incident.Transition(status, a.now()); err != nil {
		return domain.Incident{}, err
	}
	if err := a.config.Repository.UpdateIncident(incident); err != nil {
		return domain.Incident{}, err
	}
	a.record(actor, "transition", "incident", id, "incident status changed", map[string]string{"status": string(status)})
	a.emit("incident.status_changed", id, actor, map[string]any{"status": status})
	return incident, nil
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
