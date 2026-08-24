package domain

import (
	"errors"
	"strings"
	"time"
)

type StreamStatus string

const (
	StreamActive   StreamStatus = "active"
	StreamPaused   StreamStatus = "paused"
	StreamDisabled StreamStatus = "disabled"
)

type TelemetryStream struct {
	ID           string            `json:"id"`
	DeviceID     string            `json:"device_id"`
	Name         string            `json:"name"`
	Signal       string            `json:"signal"`
	Unit         string            `json:"unit"`
	SamplePeriod time.Duration     `json:"sample_period"`
	Status       StreamStatus      `json:"status"`
	Tags         map[string]string `json:"tags"`
	Version      int64             `json:"version"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func NewStream(id, deviceID, name, signal, unit string, period time.Duration, tags map[string]string, now time.Time) (TelemetryStream, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(deviceID) == "" {
		return TelemetryStream{}, errors.New("stream id and device id are required")
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(signal) == "" {
		return TelemetryStream{}, errors.New("stream name and signal are required")
	}
	if period <= 0 {
		return TelemetryStream{}, errors.New("sample period must be positive")
	}
	return TelemetryStream{
		ID: id, DeviceID: deviceID, Name: strings.TrimSpace(name),
		Signal: strings.TrimSpace(signal), Unit: strings.TrimSpace(unit),
		SamplePeriod: period, Status: StreamActive, Tags: cloneStringMap(tags),
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s TelemetryStream) AcceptsSamples() bool {
	return s.Status == StreamActive
}

func (s *TelemetryStream) Transition(status StreamStatus, now time.Time) error {
	if status == s.Status {
		return nil
	}
	if s.Status == StreamDisabled {
		return errors.New("disabled stream cannot transition")
	}
	if status != StreamActive && status != StreamPaused && status != StreamDisabled {
		return errors.New("unknown stream status")
	}
	s.Status = status
	s.Version++
	s.UpdatedAt = now
	return nil
}

func (s TelemetryStream) Clone() TelemetryStream {
	s.Tags = cloneStringMap(s.Tags)
	return s
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
