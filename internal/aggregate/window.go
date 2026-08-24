package aggregate

import (
	"errors"
	"sort"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
)

type WindowBuilder struct {
	Width time.Duration
}

func NewWindowBuilder(width time.Duration) WindowBuilder {
	if width <= 0 {
		width = 5 * time.Minute
	}
	return WindowBuilder{Width: width}
}

func (b WindowBuilder) Build(samples []domain.Sample, now time.Time) (domain.Window, error) {
	if len(samples) == 0 {
		return domain.Window{}, errors.New("no samples available")
	}
	sort.Slice(samples, func(i, j int) bool {
		return samples[i].ObservedAt.Before(samples[j].ObservedAt)
	})
	latest := samples[len(samples)-1].ObservedAt
	start := latest.Truncate(b.Width)
	end := start.Add(b.Width)
	return domain.BuildWindow(samples, start, end)
}

func (b WindowBuilder) Select(samples []domain.Sample, now time.Time) []domain.Sample {
	start := now.Add(-b.Width)
	result := make([]domain.Sample, 0, len(samples))
	for _, sample := range samples {
		if !sample.ObservedAt.Before(start) && sample.ObservedAt.Before(now.Add(time.Nanosecond)) {
			result = append(result, sample)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ObservedAt.Equal(result[j].ObservedAt) {
			return result[i].Sequence < result[j].Sequence
		}
		return result[i].ObservedAt.Before(result[j].ObservedAt)
	})
	return result
}
