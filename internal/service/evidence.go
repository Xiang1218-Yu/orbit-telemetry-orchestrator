package service

import (
	"context"
	"errors"
	"strings"

	"orbit-telemetry-orchestrator/internal/domain"
)

type EvidenceInput struct {
	ID     string              `json:"id"`
	Kind   domain.EvidenceKind `json:"kind"`
	Title  string              `json:"title"`
	Body   string              `json:"body"`
	Source string              `json:"source"`
}

func (a *App) AddEvidence(ctx context.Context, actor, incidentID string, input EvidenceInput) (domain.Evidence, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Evidence{}, err
	}
	if incident, err := a.config.Repository.GetIncident(incidentID); err != nil {
		return domain.Evidence{}, err
	} else if incident.Status == domain.IncidentClosed {
		return domain.Evidence{}, errors.New("closed incident cannot receive evidence")
	}
	evidence, err := domain.NewEvidence(input.ID, incidentID, input.Kind, input.Title, input.Body, input.Source, a.now())
	if err != nil {
		return domain.Evidence{}, err
	}
	if err := a.config.Repository.PutEvidence(evidence); err != nil {
		return domain.Evidence{}, err
	}
	if _, err := a.updateIncident(ctx, incidentID, func(incident domain.Incident) (domain.Incident, error) {
		if incident.Status == domain.IncidentClosed {
			return domain.Incident{}, errors.New("closed incident cannot receive evidence")
		}
		if err := incident.AttachEvidence(evidence.ID, a.now()); err != nil {
			return domain.Incident{}, err
		}
		return incident, nil
	}); err != nil {
		return domain.Evidence{}, err
	}
	a.record(actor, "attach", "evidence", evidence.ID, "evidence attached to incident", nil)
	a.emit("evidence.attached", evidence.ID, actor, map[string]any{"incident_id": incidentID})
	return evidence, nil
}

func (a *App) ListEvidence(ctx context.Context, incidentID string, limit int) ([]domain.Evidence, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	return a.config.Repository.ListEvidence(incidentID, limit), nil
}

func (a *App) RollupEvidence(ctx context.Context, incidentID string) (domain.Evidence, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Evidence{}, err
	}
	incident, err := a.config.Repository.GetIncident(incidentID)
	if err != nil {
		return domain.Evidence{}, err
	}
	title := "Automatic incident snapshot"
	body := "incident=" + incident.ID + "; anomalies=" + itoa(len(incident.AnomalyIDs)) + "; actions=" + itoa(len(incident.ActionIDs))
	evidence, err := domain.NewEvidence(a.nextID("evidence"), incidentID, domain.EvidenceSnapshot, title, body, "orbit-rollup", a.now())
	if err != nil {
		return domain.Evidence{}, err
	}
	if err := a.config.Repository.PutEvidence(evidence); err != nil {
		return domain.Evidence{}, err
	}
	_, _ = a.updateIncident(ctx, incidentID, func(current domain.Incident) (domain.Incident, error) {
		_ = current.AttachEvidence(evidence.ID, a.now())
		return current, nil
	})
	return evidence, nil
}

func (a *App) EvidenceSummary(ctx context.Context, incidentID string) (map[string]int, error) {
	values, err := a.ListEvidence(ctx, incidentID, 0)
	if err != nil {
		return nil, err
	}
	summary := make(map[string]int)
	for _, value := range values {
		summary[string(value.Kind)]++
	}
	return summary, nil
}

func evidenceText(value domain.Evidence) string {
	return strings.TrimSpace(value.Title) + ": " + strings.TrimSpace(value.Body)
}
