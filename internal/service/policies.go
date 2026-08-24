package service

import (
	"context"
	"errors"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/store"
)

type PolicyInput struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	DeviceID      string              `json:"device_id"`
	Version       int                 `json:"version"`
	EffectiveFrom time.Time           `json:"effective_from"`
	Rules         []domain.Rule       `json:"rules"`
	Response      domain.ResponsePlan `json:"response"`
}

func (a *App) CreatePolicy(ctx context.Context, actor string, input PolicyInput) (domain.CollectionPolicy, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.CollectionPolicy{}, err
	}
	device, err := a.config.Repository.GetDevice(input.DeviceID)
	if err != nil {
		return domain.CollectionPolicy{}, err
	}
	if device.Status == domain.DeviceRetired {
		return domain.CollectionPolicy{}, errors.New("retired device cannot receive policy")
	}
	policy, err := domain.NewPolicy(input.ID, input.Name, input.DeviceID, actor, input.Version, input.Rules, input.Response, input.EffectiveFrom, a.now())
	if err != nil {
		return domain.CollectionPolicy{}, err
	}
	if err := a.config.Repository.PutPolicy(policy); err != nil {
		return domain.CollectionPolicy{}, err
	}
	a.record(actor, "create", "policy", policy.ID, "collection policy created", nil)
	return policy, nil
}

func (a *App) GetPolicy(ctx context.Context, id string) (domain.CollectionPolicy, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.CollectionPolicy{}, err
	}
	return a.config.Repository.GetPolicy(id)
}

func (a *App) ListPolicies(ctx context.Context, deviceID string, status domain.PolicyStatus, limit int) ([]domain.CollectionPolicy, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	return a.config.Repository.ListPolicies(deviceID, status, limit), nil
}

func (a *App) PublishPolicy(ctx context.Context, actor, id string) (domain.CollectionPolicy, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.CollectionPolicy{}, err
	}
	policy, err := a.config.Repository.GetPolicy(id)
	if err != nil {
		return domain.CollectionPolicy{}, err
	}
	if current, currentErr := a.config.Repository.ActivePolicy(policy.DeviceID, a.now()); currentErr == nil && current.Version >= policy.Version {
		return domain.CollectionPolicy{}, errors.New("policy version must exceed active version")
	}
	if err := policy.Publish(a.now()); err != nil {
		return domain.CollectionPolicy{}, err
	}
	if err := a.config.Repository.UpdatePolicy(policy, policy.Version); err != nil {
		return domain.CollectionPolicy{}, err
	}
	a.record(actor, "publish", "policy", id, "collection policy published", nil)
	a.emit("policy.published", id, actor, map[string]any{"device_id": policy.DeviceID, "version": policy.Version})
	return policy, nil
}

func (a *App) LockPolicy(ctx context.Context, actor, id string) (domain.CollectionPolicy, error) {
	policy, err := a.GetPolicy(ctx, id)
	if err != nil {
		return domain.CollectionPolicy{}, err
	}
	if err := policy.Lock(a.now()); err != nil {
		return domain.CollectionPolicy{}, err
	}
	if err := a.config.Repository.UpdatePolicy(policy, policy.Version); err != nil {
		return domain.CollectionPolicy{}, err
	}
	a.record(actor, "lock", "policy", id, "policy locked", nil)
	return policy, nil
}

func (a *App) ActivePolicy(ctx context.Context, deviceID string) (domain.CollectionPolicy, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.CollectionPolicy{}, err
	}
	return a.config.Repository.ActivePolicy(deviceID, a.now())
}

var _ = store.ErrExists
