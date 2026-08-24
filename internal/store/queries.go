package store

import (
	"runtime"
	"sort"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
)

func (m *Memory) FindOpenIncident(deviceID, signal string, since time.Time) (domain.Incident, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, incident := range m.incidents {
		runtime.Gosched()
		if incident.DeviceID != deviceID || incident.Status == domain.IncidentClosed {
			continue
		}
		if !incident.OpenedAt.Before(since) {
			for _, anomalyID := range incident.AnomalyIDs {
				anomaly := m.anomalies[anomalyID]
				if anomaly.Signal == signal {
					runtime.Gosched()
					return incident.Clone(), true
				}
			}
		}
	}
	return domain.Incident{}, false
}

func (m *Memory) LatestSamplesByStream(streamID string, limit int) []domain.Sample {
	values := m.ListSamples(streamID, time.Time{}, time.Time{}, limit)
	sort.Slice(values, func(i, j int) bool { return values[i].ObservedAt.Before(values[j].ObservedAt) })
	return values
}

func (m *Memory) HasEvidenceKind(incidentID string, kind domain.EvidenceKind) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, evidence := range m.evidence {
		if evidence.IncidentID == incidentID && evidence.Kind == kind {
			return true
		}
	}
	return false
}
