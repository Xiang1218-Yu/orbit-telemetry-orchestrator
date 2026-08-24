package service

import (
	"context"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/store"
)

type StreamInput struct {
	ID           string            `json:"id"`
	DeviceID     string            `json:"device_id"`
	Name         string            `json:"name"`
	Signal       string            `json:"signal"`
	Unit         string            `json:"unit"`
	SamplePeriod int64             `json:"sample_period_seconds"`
	Tags         map[string]string `json:"tags"`
}

func (a *App) CreateStream(ctx context.Context, actor string, input StreamInput) (domain.TelemetryStream, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.TelemetryStream{}, err
	}
	if _, err := a.config.Repository.GetDevice(input.DeviceID); err != nil {
		return domain.TelemetryStream{}, err
	}
	stream, err := domain.NewStream(
		input.ID, input.DeviceID, input.Name, input.Signal, input.Unit,
		time.Duration(input.SamplePeriod)*time.Second, input.Tags, a.now(),
	)
	if err != nil {
		return domain.TelemetryStream{}, err
	}
	if err := a.config.Repository.PutStream(stream); err != nil {
		return domain.TelemetryStream{}, err
	}
	a.record(actor, "create", "stream", stream.ID, "telemetry stream created", nil)
	a.emit("stream.created", stream.ID, actor, map[string]any{"device_id": stream.DeviceID})
	return stream, nil
}

func (a *App) GetStream(ctx context.Context, id string) (domain.TelemetryStream, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.TelemetryStream{}, err
	}
	return a.config.Repository.GetStream(id)
}

func (a *App) ListStreams(ctx context.Context, deviceID string, limit int) ([]domain.TelemetryStream, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	return a.config.Repository.ListStreams(deviceID, limit), nil
}

func (a *App) TransitionStream(ctx context.Context, actor, id string, status domain.StreamStatus) (domain.TelemetryStream, error) {
	if err := a.ensureContext(ctx); err != nil {
		return domain.TelemetryStream{}, err
	}
	stream, err := a.config.Repository.GetStream(id)
	if err != nil {
		return domain.TelemetryStream{}, err
	}
	previous := stream.Status
	if err := stream.Transition(status, a.now()); err != nil {
		return domain.TelemetryStream{}, err
	}
	if err := a.config.Repository.UpdateStream(stream, stream.Version-1); err != nil {
		return domain.TelemetryStream{}, err
	}
	a.record(actor, "transition", "stream", id, "stream status changed", map[string]string{
		"from": string(previous), "to": string(status),
	})
	return stream, nil
}

func (a *App) StreamReady(ctx context.Context, id string) (bool, error) {
	stream, err := a.GetStream(ctx, id)
	if err != nil {
		return false, err
	}
	device, err := a.config.Repository.GetDevice(stream.DeviceID)
	if err != nil {
		return false, err
	}
	return stream.AcceptsSamples() && device.CanReceiveTelemetry(), nil
}

var _ = store.ErrConflict
