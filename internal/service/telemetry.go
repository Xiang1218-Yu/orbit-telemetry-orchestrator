package service

import (
	"context"
	"errors"
	"time"

	"orbit-telemetry-orchestrator/internal/aggregate"
	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/store"
)

type TelemetryBatch struct {
	Samples []TelemetryInput `json:"samples"`
}

type TelemetryInput struct {
	StreamID   string    `json:"stream_id"`
	Value      float64   `json:"value"`
	Quality    string    `json:"quality"`
	ObservedAt time.Time `json:"observed_at"`
	Sequence   int64     `json:"sequence"`
}

func (a *App) IngestTelemetry(ctx context.Context, actor string, batch TelemetryBatch) (int, error) {
	if err := a.ensureContext(ctx); err != nil {
		return 0, err
	}
	if len(batch.Samples) == 0 {
		return 0, errors.New("telemetry batch is empty")
	}
	now := a.now()
	result := make([]domain.Sample, 0, len(batch.Samples))
	streams := make(map[string]domain.TelemetryStream)
	for _, input := range batch.Samples {
		stream, ok := streams[input.StreamID]
		if !ok {
			var err error
			stream, err = a.config.Repository.GetStream(input.StreamID)
			if err != nil {
				return 0, err
			}
			streams[input.StreamID] = stream
		}
		device, err := a.config.Repository.GetDevice(stream.DeviceID)
		if err != nil {
			return 0, err
		}
		if !stream.AcceptsSamples() || !device.CanReceiveTelemetry() {
			return 0, errors.New("stream or device is not ready for telemetry")
		}
		sample, err := domain.NewSample(stream.ID, stream.DeviceID, stream.Signal, input.Quality, input.Value, input.ObservedAt, now, input.Sequence)
		if err != nil {
			return 0, err
		}
		result = append(result, sample)
	}
	a.config.Repository.AddSamples(result)
	for _, stream := range streams {
		_ = a.TouchDevice(ctx, stream.DeviceID, timeValue{Time: now})
		a.submit("incident-evaluation", IncidentEvaluation{StreamID: stream.ID, DeviceID: stream.DeviceID})
	}
	a.record(actor, "ingest", "telemetry", "", "telemetry batch ingested", map[string]string{"samples": itoa(len(result))})
	a.config.Metrics.Add("orbit_samples_ingested_total", uint64(len(result)), nil)
	return len(result), nil
}

func (a *App) EvaluateTelemetry(ctx context.Context, streamID, deviceID string) ([]domain.Anomaly, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	policy, err := a.config.Repository.ActivePolicy(deviceID, a.now())
	if err != nil {
		return nil, nil
	}
	stream, err := a.config.Repository.GetStream(streamID)
	if err != nil {
		return nil, err
	}
	start, end := aggregate.WindowBounds(a.now(), 5*time.Minute)
	samples := a.config.Repository.ListSamples(streamID, start, end, 500)
	window, err := domain.BuildWindow(samples, start, end)
	if err != nil {
		return nil, nil
	}
	detections := aggregate.Detect(window, policy.Rules)
	result := make([]domain.Anomaly, 0, len(detections))
	for _, detection := range detections {
		anomalyID := a.nextID("anomaly")
		anomaly := domain.Anomaly{
			ID: anomalyID, DeviceID: deviceID, StreamID: streamID, Signal: stream.Signal,
			PolicyID: policy.ID, RuleIndex: detection.RuleIndex, Severity: detection.Severity,
			Observed: window.LastValue, Threshold: detection.Threshold, Window: window,
			Status: domain.AnomalyOpen, CreatedAt: a.now(), UpdatedAt: a.now(),
		}
		if err := a.config.Repository.PutAnomaly(anomaly); err == store.ErrExists {
			continue
		} else if err != nil {
			return result, err
		}
		result = append(result, anomaly)
		a.config.Metrics.Inc("orbit_anomalies_created_total", map[string]string{"severity": itoa(detection.Severity)})
		a.emit("anomaly.detected", anomaly.ID, "system", map[string]any{"device_id": deviceID, "severity": detection.Severity})
		if incident, ok := a.correlateIncident(ctx, anomaly); ok {
			anomaly.IncidentID = incident.ID
			_ = a.config.Repository.UpdateAnomaly(anomaly)
		}
	}
	return result, nil
}

func (a *App) Samples(ctx context.Context, streamID string, limit int) ([]domain.Sample, error) {
	if err := a.ensureContext(ctx); err != nil {
		return nil, err
	}
	return a.config.Repository.LatestSamplesByStream(streamID, limit), nil
}

func (a *App) nextID(prefix string) string {
	a.sequence++
	return prefix + "-" + itoa(int(a.sequence))
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	return sign + string(buffer[index:])
}
