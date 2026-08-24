package service

import (
	"context"
	"errors"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/store"
)

type ActionInput struct {
	ID          string            `json:"id"`
	IncidentID  string            `json:"incident_id"`
	Type        domain.ActionType `json:"type"`
	MaxAttempts int               `json:"max_attempts"`
	Idempotency string            `json:"idempotency"`
	Reason      string            `json:"reason"`
}

func (a *App) CreateAction(ctx context.Context, actor string, input ActionInput) (domain.ResponseAction, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.ResponseAction{}, err
	}
	incident, err := a.config.Repository.GetIncident(input.IncidentID)
	if err != nil {
		return domain.ResponseAction{}, err
	}
	action, err := domain.NewAction(
		input.ID, input.IncidentID, incident.DeviceID, input.Type, input.MaxAttempts,
		input.Idempotency, input.Reason, a.now(),
	)
	if err != nil {
		return domain.ResponseAction{}, err
	}
	if err := a.config.Repository.PutAction(action); err != nil {
		return domain.ResponseAction{}, err
	}
	_ = incident.AttachAction(action.ID, a.now())
	_ = a.config.Repository.UpdateIncident(incident)
	a.record(actor, "create", "action", action.ID, "response action queued", nil)
	a.submit("response-action", ResponseActionJob{ActionID: action.ID})
	return action, nil
}

func (a *App) GetAction(ctx context.Context, id string) (domain.ResponseAction, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.ResponseAction{}, err
	}
	return a.config.Repository.GetAction(id)
}

func (a *App) ListActions(ctx context.Context, incidentID, status string, limit int) ([]domain.ResponseAction, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	return a.config.Repository.ListActions(incidentID, status, limit), nil
}

func (a *App) ExecuteAction(ctx context.Context, id string) (domain.ResponseAction, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.ResponseAction{}, err
	}
	action, err := a.config.Repository.GetAction(id)
	if err != nil {
		return domain.ResponseAction{}, err
	}
	if action.Status == domain.ActionSucceeded {
		return action, nil
	}
	if err := action.Start(a.now()); err != nil {
		return domain.ResponseAction{}, err
	}
	if err := a.config.Repository.UpdateAction(action); err != nil {
		return domain.ResponseAction{}, err
	}
	success, result := simulateAction(action)
	action.Finish(success, result, a.now())
	if err := a.config.Repository.UpdateAction(action); err != nil {
		return domain.ResponseAction{}, err
	}
	a.record("system", "execute", "action", action.ID, result, nil)
	a.config.Metrics.Inc("orbit_actions_finished_total", map[string]string{"status": string(action.Status)})
	if !success && action.CanRetry() {
		a.submit("response-action", ResponseActionJob{ActionID: action.ID})
	}
	return action, nil
}

func simulateAction(action domain.ResponseAction) (bool, string) {
	switch action.Type {
	case domain.ActionNotify:
		return true, "notification delivered"
	case domain.ActionCollect:
		return true, "diagnostic collection requested"
	case domain.ActionThrottle:
		return true, "device telemetry rate reduced"
	case domain.ActionIsolate:
		return true, "device isolated from production traffic"
	case domain.ActionRestart:
		if action.Attempt == 1 {
			return false, "restart request timed out"
		}
		return true, "agent restarted"
	default:
		return false, "unsupported action"
	}
}

func (a *App) CancelAction(ctx context.Context, actor, id string) (domain.ResponseAction, error) {
	action, err := a.GetAction(ctx, id)
	if err != nil {
		return domain.ResponseAction{}, err
	}
	if action.Status != domain.ActionQueued {
		return domain.ResponseAction{}, errors.New("only queued actions can be cancelled")
	}
	action.Status = domain.ActionSkipped
	action.Result = "cancelled by operator"
	action.UpdatedAt = a.now()
	if err := a.config.Repository.UpdateAction(action); err != nil {
		return domain.ResponseAction{}, err
	}
	a.record(actor, "cancel", "action", id, "response action cancelled", nil)
	return action, nil
}

var _ = store.ErrConflict
