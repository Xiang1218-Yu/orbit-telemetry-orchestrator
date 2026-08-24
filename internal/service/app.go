package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"orbit-telemetry-orchestrator/internal/audit"
	"orbit-telemetry-orchestrator/internal/clock"
	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/events"
	"orbit-telemetry-orchestrator/internal/metrics"
	"orbit-telemetry-orchestrator/internal/queue"
	"orbit-telemetry-orchestrator/internal/store"
)

type Config struct {
	Repository *store.Memory
	Events     *events.Bus
	Audit      *audit.Log
	Queue      *queue.Queue
	Metrics    *metrics.Registry
	Clock      clock.Clock
	Logger     *slog.Logger
}

type App struct {
	config   Config
	sequence uint64
}

func New(config Config) *App {
	if config.Repository == nil {
		config.Repository = store.NewMemory()
	}
	if config.Events == nil {
		config.Events = events.NewBus(events.Config{Buffer: 32, DropWhenFull: true})
	}
	if config.Audit == nil {
		config.Audit = audit.NewLog(1000)
	}
	if config.Metrics == nil {
		config.Metrics = metrics.NewRegistry()
	}
	if config.Clock == nil {
		config.Clock = clock.Real{}
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	return &App{config: config}
}

func (a *App) now() time.Time {
	return a.config.Clock.Now().UTC()
}

func (a *App) ensureContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (a *App) emit(eventType, subject, actor string, payload map[string]any) {
	a.config.Events.Publish(a.config.Events.NewEvent(eventType, subject, actor, payload, a.now()))
}

func (a *App) record(actor, action, resource, resourceID, message string, changes map[string]string) {
	a.config.Audit.Record(actor, action, resource, resourceID, message, changes, a.now())
}

func (a *App) submit(jobType string, payload any) {
	if a.config.Queue == nil {
		return
	}
	if _, err := a.config.Queue.Submit(jobType, payload); err != nil {
		a.config.Logger.Warn("background job not queued", "type", jobType, "error", err)
	}
}

func (a *App) Stats(ctx context.Context) (store.Counts, error) {
	if err := a.ensureContext(ctx); err != nil {
		return store.Counts{}, err
	}
	return a.config.Repository.Counts(a.now()), nil
}

func (a *App) Seed() {
	now := a.now()
	device, err := domain.NewDevice("device-demo-1", "Demo Edge Gateway", "lab-a", "orbit-gw", []string{"demo", "edge"}, now)
	if err == nil {
		_ = device.Transition(domain.DeviceReady, now)
		_ = a.config.Repository.PutDevice(device)
	}
}

func (a *App) ProcessIncidentEvaluation(ctx context.Context, job queue.Job) error {
	payload, ok := job.Payload.(IncidentEvaluation)
	if !ok {
		return errors.New("invalid incident evaluation payload")
	}
	if err := a.ensureContext(ctx); err != nil {
		return err
	}
	_, err := a.EvaluateTelemetry(ctx, payload.StreamID, payload.DeviceID)
	return err
}

func (a *App) ProcessResponseAction(ctx context.Context, job queue.Job) error {
	payload, ok := job.Payload.(ResponseActionJob)
	if !ok {
		return errors.New("invalid response action payload")
	}
	_, err := a.ExecuteAction(ctx, payload.ActionID)
	return err
}

func (a *App) ProcessEvidenceRollup(ctx context.Context, job queue.Job) error {
	payload, ok := job.Payload.(EvidenceRollupJob)
	if !ok {
		return errors.New("invalid evidence rollup payload")
	}
	_, err := a.RollupEvidence(ctx, payload.IncidentID)
	return err
}

type IncidentEvaluation struct {
	StreamID string
	DeviceID string
}

type ResponseActionJob struct {
	ActionID string
}

type EvidenceRollupJob struct {
	IncidentID string
}
