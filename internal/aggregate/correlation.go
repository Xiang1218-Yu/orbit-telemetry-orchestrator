package aggregate

import (
	"sort"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
)

type Correlator struct {
	Window time.Duration
}

func NewCorrelator(window time.Duration) Correlator {
	if window <= 0 {
		window = 10 * time.Minute
	}
	return Correlator{Window: window}
}

func (c Correlator) ShouldMerge(incident domain.Incident, anomaly domain.Anomaly, now time.Time) bool {
	if incident.DeviceID != anomaly.DeviceID || incident.Status == domain.IncidentClosed {
		return false
	}
	return !anomaly.CreatedAt.Before(now.Add(-c.Window))
}

func SortAnomalies(values []domain.Anomaly) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Severity == values[j].Severity {
			return values[i].CreatedAt.Before(values[j].CreatedAt)
		}
		return values[i].Severity > values[j].Severity
	})
}
