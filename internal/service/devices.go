package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/store"
)

type DeviceInput struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Site   string   `json:"site"`
	Model  string   `json:"model"`
	Labels []string `json:"labels"`
}

func (a *App) CreateDevice(ctx context.Context, actor string, input DeviceInput) (domain.Device, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Device{}, err
	}
	// A cancelled request must not mutate system state: skip device creation,
	// audit records, events, and metrics so no half-finished record survives.
	if err := ctx.Err(); err != nil {
		return domain.Device{}, err
	}
	device, err := domain.NewDevice(input.ID, input.Name, input.Site, input.Model, input.Labels, a.now())
	if err != nil {
		return domain.Device{}, err
	}
	if err := a.config.Repository.PutDevice(device); err != nil {
		return domain.Device{}, err
	}
	// Re-check after the write so a cancellation that raced the persistence step
	// does not leave a half-finished record blocking future registration.
	if err := ctx.Err(); err != nil {
		_ = a.config.Repository.DeleteDevice(device.ID)
		return domain.Device{}, err
	}
	a.record(actor, "create", "device", device.ID, "device registered", nil)
	a.emit("device.registered", device.ID, actor, map[string]any{"device_id": device.ID})
	a.config.Metrics.Inc("orbit_devices_created_total", nil)
	return device, nil
}

func (a *App) GetDevice(ctx context.Context, id string) (domain.Device, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Device{}, err
	}
	return a.config.Repository.GetDevice(strings.TrimSpace(id))
}

func (a *App) ListDevices(ctx context.Context, status domain.DeviceStatus, limit int) ([]domain.Device, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	return a.config.Repository.ListDevices(status, limit), nil
}

func (a *App) TransitionDevice(ctx context.Context, actor, id string, status domain.DeviceStatus) (domain.Device, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.Device{}, err
	}
	device, err := a.config.Repository.GetDevice(id)
	if err != nil {
		return domain.Device{}, err
	}
	previous := device.Status
	if err := device.Transition(status, a.now()); err != nil {
		return domain.Device{}, err
	}
	if err := a.config.Repository.UpdateDevice(device, device.Version-1); err != nil {
		return domain.Device{}, err
	}
	a.record(actor, "transition", "device", id, "device status changed", map[string]string{
		"from": string(previous), "to": string(status),
	})
	a.emit("device.status_changed", id, actor, map[string]any{"status": status})
	return device, nil
}

func (a *App) TouchDevice(ctx context.Context, id string, observedAt timeValue) error {
	if err := a.ensureContext(ctx); err != nil {
		return err
	}
	device, err := a.config.Repository.GetDevice(id)
	if err != nil {
		return err
	}
	if device.Status == domain.DeviceRetired {
		return errors.New("retired device cannot be touched")
	}
	now := observedAt.Time
	device.LastSeenAt = &now
	device.FailureCount = 0
	device.Version++
	device.UpdatedAt = a.now()
	return a.config.Repository.UpdateDevice(device, device.Version-1)
}

type timeValue struct {
	Time time.Time
}

var _ = store.ErrNotFound
