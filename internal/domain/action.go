package domain

import (
	"errors"
	"time"
)

type ActionType string

const (
	ActionNotify   ActionType = "notify"
	ActionIsolate  ActionType = "isolate"
	ActionThrottle ActionType = "throttle"
	ActionCollect  ActionType = "collect-diagnostics"
	ActionRestart  ActionType = "restart-agent"
)

type ActionStatus string

const (
	ActionQueued    ActionStatus = "queued"
	ActionRunning   ActionStatus = "running"
	ActionSucceeded ActionStatus = "succeeded"
	ActionFailed    ActionStatus = "failed"
	ActionSkipped   ActionStatus = "skipped"
)

type ResponseAction struct {
	ID          string       `json:"id"`
	IncidentID  string       `json:"incident_id"`
	DeviceID    string       `json:"device_id"`
	Type        ActionType   `json:"type"`
	Status      ActionStatus `json:"status"`
	Attempt     int          `json:"attempt"`
	MaxAttempts int          `json:"max_attempts"`
	Idempotency string       `json:"idempotency"`
	Reason      string       `json:"reason"`
	Result      string       `json:"result,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	FinishedAt  *time.Time   `json:"finished_at,omitempty"`
}

func NewAction(id, incidentID, deviceID string, actionType ActionType, maxAttempts int, key, reason string, now time.Time) (ResponseAction, error) {
	if id == "" || incidentID == "" || deviceID == "" || key == "" {
		return ResponseAction{}, errors.New("action identity fields are required")
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	switch actionType {
	case ActionNotify, ActionIsolate, ActionThrottle, ActionCollect, ActionRestart:
	default:
		return ResponseAction{}, errors.New("unknown action type")
	}
	return ResponseAction{
		ID: id, IncidentID: incidentID, DeviceID: deviceID, Type: actionType,
		Status: ActionQueued, MaxAttempts: maxAttempts, Idempotency: key,
		Reason: reason, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (a ResponseAction) Clone() ResponseAction {
	if a.FinishedAt != nil {
		copy := *a.FinishedAt
		a.FinishedAt = &copy
	}
	return a
}

func (a *ResponseAction) Start(now time.Time) error {
	if a.Status != ActionQueued && a.Status != ActionFailed {
		return errors.New("action is not ready to start")
	}
	if a.Attempt >= a.MaxAttempts {
		return errors.New("action exhausted retries")
	}
	a.Attempt++
	a.Status = ActionRunning
	a.UpdatedAt = now
	return nil
}

func (a *ResponseAction) Finish(success bool, result string, now time.Time) {
	a.Result = result
	a.UpdatedAt = now
	copy := now
	a.FinishedAt = &copy
	if success {
		a.Status = ActionSucceeded
	} else {
		a.Status = ActionFailed
	}
}

func (a ResponseAction) CanRetry() bool {
	return a.Status == ActionFailed && a.Attempt < a.MaxAttempts
}
