package domain

import (
	"errors"
	"math"
	"strings"
	"time"
)

type Sample struct {
	StreamID   string    `json:"stream_id"`
	DeviceID   string    `json:"device_id"`
	Signal     string    `json:"signal"`
	Value      float64   `json:"value"`
	Quality    string    `json:"quality"`
	ObservedAt time.Time `json:"observed_at"`
	ReceivedAt time.Time `json:"received_at"`
	Sequence   int64     `json:"sequence"`
}

func NewSample(streamID, deviceID, signal, quality string, value float64, observedAt, receivedAt time.Time, sequence int64) (Sample, error) {
	if strings.TrimSpace(streamID) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(signal) == "" {
		return Sample{}, errors.New("sample stream, device, and signal are required")
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return Sample{}, errors.New("sample value must be finite")
	}
	if observedAt.IsZero() || receivedAt.IsZero() {
		return Sample{}, errors.New("sample timestamps are required")
	}
	if sequence < 1 {
		return Sample{}, errors.New("sample sequence must be positive")
	}
	if quality == "" {
		quality = "good"
	}
	return Sample{
		StreamID: streamID, DeviceID: deviceID, Signal: signal, Value: value,
		Quality: quality, ObservedAt: observedAt.UTC(), ReceivedAt: receivedAt.UTC(),
		Sequence: sequence,
	}, nil
}

type Window struct {
	DeviceID   string    `json:"device_id"`
	StreamID   string    `json:"stream_id"`
	Signal     string    `json:"signal"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
	Count      int       `json:"count"`
	Min        float64   `json:"min"`
	Max        float64   `json:"max"`
	Average    float64   `json:"average"`
	LastValue  float64   `json:"last_value"`
	BadSamples int       `json:"bad_samples"`
}

func BuildWindow(samples []Sample, start, end time.Time) (Window, error) {
	if len(samples) == 0 {
		return Window{}, errors.New("cannot build window without samples")
	}
	if !end.After(start) {
		return Window{}, errors.New("window end must be after start")
	}
	result := Window{
		DeviceID: samples[0].DeviceID, StreamID: samples[0].StreamID,
		Signal: samples[0].Signal, StartedAt: start.UTC(), EndedAt: end.UTC(),
		Min: samples[0].Value, Max: samples[0].Value,
	}
	var total float64
	for _, sample := range samples {
		if sample.DeviceID != result.DeviceID || sample.StreamID != result.StreamID || sample.Signal != result.Signal {
			return Window{}, errors.New("window samples must belong to one stream")
		}
		if sample.ObservedAt.Before(start) || !sample.ObservedAt.Before(end) {
			continue
		}
		result.Count++
		total += sample.Value
		result.LastValue = sample.Value
		if sample.Value < result.Min {
			result.Min = sample.Value
		}
		if sample.Value > result.Max {
			result.Max = sample.Value
		}
		if sample.Quality != "good" {
			result.BadSamples++
		}
	}
	if result.Count == 0 {
		return Window{}, errors.New("window contains no samples")
	}
	result.Average = total / float64(result.Count)
	return result, nil
}
