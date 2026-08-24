package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
)

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("record version conflict")
var ErrExists = errors.New("record already exists")

type Memory struct {
	mu          sync.RWMutex
	devices     map[string]domain.Device
	streams     map[string]domain.TelemetryStream
	policies    map[string]domain.CollectionPolicy
	samples     []domain.Sample
	anomalies   map[string]domain.Anomaly
	incidents   map[string]domain.Incident
	actions     map[string]domain.ResponseAction
	evidence    map[string]domain.Evidence
	idempotency map[string]string
}

func NewMemory() *Memory {
	return &Memory{
		devices: make(map[string]domain.Device), streams: make(map[string]domain.TelemetryStream),
		policies: make(map[string]domain.CollectionPolicy), anomalies: make(map[string]domain.Anomaly),
		incidents: make(map[string]domain.Incident), actions: make(map[string]domain.ResponseAction),
		evidence: make(map[string]domain.Evidence), idempotency: make(map[string]string),
		samples: make([]domain.Sample, 0, 1024),
	}
}

func (m *Memory) PutDevice(device domain.Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.devices[device.ID]; ok {
		return ErrExists
	}
	m.devices[device.ID] = device.Clone()
	return nil
}

func (m *Memory) GetDevice(id string) (domain.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	device, ok := m.devices[id]
	if !ok {
		return domain.Device{}, ErrNotFound
	}
	return device.Clone(), nil
}

func (m *Memory) UpdateDevice(device domain.Device, expected int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.devices[device.ID]
	if !ok {
		return ErrNotFound
	}
	if expected > 0 && current.Version != expected {
		return ErrConflict
	}
	m.devices[device.ID] = device.Clone()
	return nil
}

func (m *Memory) ListDevices(status domain.DeviceStatus, limit int) []domain.Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Device, 0, len(m.devices))
	for _, device := range m.devices {
		if status != "" && device.Status != status {
			continue
		}
		result = append(result, device.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return capList(result, limit)
}

func (m *Memory) PutStream(stream domain.TelemetryStream) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.streams[stream.ID]; ok {
		return ErrExists
	}
	m.streams[stream.ID] = stream.Clone()
	return nil
}

func (m *Memory) GetStream(id string) (domain.TelemetryStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stream, ok := m.streams[id]
	if !ok {
		return domain.TelemetryStream{}, ErrNotFound
	}
	return stream.Clone(), nil
}

func (m *Memory) UpdateStream(stream domain.TelemetryStream, expected int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.streams[stream.ID]
	if !ok {
		return ErrNotFound
	}
	if expected > 0 && current.Version != expected {
		return ErrConflict
	}
	m.streams[stream.ID] = stream.Clone()
	return nil
}

func (m *Memory) ListStreams(deviceID string, limit int) []domain.TelemetryStream {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.TelemetryStream, 0, len(m.streams))
	for _, stream := range m.streams {
		if deviceID != "" && stream.DeviceID != deviceID {
			continue
		}
		result = append(result, stream.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return capList(result, limit)
}

func (m *Memory) PutPolicy(policy domain.CollectionPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.policies[policy.ID]; ok {
		return ErrExists
	}
	m.policies[policy.ID] = policy.Clone()
	return nil
}

func (m *Memory) GetPolicy(id string) (domain.CollectionPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	policy, ok := m.policies[id]
	if !ok {
		return domain.CollectionPolicy{}, ErrNotFound
	}
	return policy.Clone(), nil
}

func (m *Memory) UpdatePolicy(policy domain.CollectionPolicy, expected int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.policies[policy.ID]
	if !ok {
		return ErrNotFound
	}
	if expected > 0 && current.Version != expected {
		return ErrConflict
	}
	m.policies[policy.ID] = policy.Clone()
	return nil
}

func (m *Memory) ListPolicies(deviceID string, status domain.PolicyStatus, limit int) []domain.CollectionPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.CollectionPolicy, 0, len(m.policies))
	for _, policy := range m.policies {
		if deviceID != "" && policy.DeviceID != deviceID {
			continue
		}
		if status != "" && policy.Status != status {
			continue
		}
		result = append(result, policy.Clone())
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].DeviceID == result[j].DeviceID {
			return result[i].Version > result[j].Version
		}
		return result[i].DeviceID < result[j].DeviceID
	})
	return capList(result, limit)
}

func (m *Memory) ActivePolicy(deviceID string, now time.Time) (domain.CollectionPolicy, error) {
	policies := m.ListPolicies(deviceID, "", 0)
	for _, policy := range policies {
		if policy.ActiveAt(now) {
			return policy, nil
		}
	}
	return domain.CollectionPolicy{}, ErrNotFound
}

func (m *Memory) AddSamples(samples []domain.Sample) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.samples = append(m.samples, samples...)
	if len(m.samples) > 10000 {
		m.samples = append([]domain.Sample(nil), m.samples[len(m.samples)-10000:]...)
	}
}

func (m *Memory) ListSamples(streamID string, start, end time.Time, limit int) []domain.Sample {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Sample, 0)
	for index := len(m.samples) - 1; index >= 0; index-- {
		sample := m.samples[index]
		if streamID != "" && sample.StreamID != streamID {
			continue
		}
		if !start.IsZero() && sample.ObservedAt.Before(start) {
			continue
		}
		if !end.IsZero() && !sample.ObservedAt.Before(end) {
			continue
		}
		result = append(result, sample)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func (m *Memory) PutAnomaly(anomaly domain.Anomaly) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.anomalies[anomaly.ID]; ok {
		return ErrExists
	}
	m.anomalies[anomaly.ID] = anomaly.Clone()
	return nil
}

func (m *Memory) GetAnomaly(id string) (domain.Anomaly, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	anomaly, ok := m.anomalies[id]
	if !ok {
		return domain.Anomaly{}, ErrNotFound
	}
	return anomaly.Clone(), nil
}

func (m *Memory) UpdateAnomaly(anomaly domain.Anomaly) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.anomalies[anomaly.ID]; !ok {
		return ErrNotFound
	}
	m.anomalies[anomaly.ID] = anomaly.Clone()
	return nil
}

func (m *Memory) ListAnomalies(deviceID, status string, limit int) []domain.Anomaly {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Anomaly, 0, len(m.anomalies))
	for _, anomaly := range m.anomalies {
		if deviceID != "" && anomaly.DeviceID != deviceID {
			continue
		}
		if status != "" && string(anomaly.Status) != status {
			continue
		}
		result = append(result, anomaly.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return capList(result, limit)
}

func (m *Memory) PutIncident(incident domain.Incident) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.incidents[incident.ID]; ok {
		return ErrExists
	}
	m.incidents[incident.ID] = incident.Clone()
	return nil
}

func (m *Memory) GetIncident(id string) (domain.Incident, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	incident, ok := m.incidents[id]
	if !ok {
		return domain.Incident{}, ErrNotFound
	}
	return incident.Clone(), nil
}

func (m *Memory) UpdateIncident(incident domain.Incident) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.incidents[incident.ID]; !ok {
		return ErrNotFound
	}
	m.incidents[incident.ID] = incident.Clone()
	return nil
}

func (m *Memory) ListIncidents(deviceID, status string, limit int) []domain.Incident {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Incident, 0, len(m.incidents))
	for _, incident := range m.incidents {
		if deviceID != "" && incident.DeviceID != deviceID {
			continue
		}
		if status != "" && string(incident.Status) != status {
			continue
		}
		result = append(result, incident.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return capList(result, limit)
}

func (m *Memory) PutAction(action domain.ResponseAction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.idempotency[action.Idempotency]; ok {
		if existing != action.ID {
			return ErrConflict
		}
		return ErrExists
	}
	if _, ok := m.actions[action.ID]; ok {
		return ErrExists
	}
	m.actions[action.ID] = action.Clone()
	m.idempotency[action.Idempotency] = action.ID
	return nil
}

func (m *Memory) GetAction(id string) (domain.ResponseAction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	action, ok := m.actions[id]
	if !ok {
		return domain.ResponseAction{}, ErrNotFound
	}
	return action.Clone(), nil
}

func (m *Memory) UpdateAction(action domain.ResponseAction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.actions[action.ID]; !ok {
		return ErrNotFound
	}
	m.actions[action.ID] = action.Clone()
	return nil
}

func (m *Memory) ListActions(incidentID, status string, limit int) []domain.ResponseAction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.ResponseAction, 0, len(m.actions))
	for _, action := range m.actions {
		if incidentID != "" && action.IncidentID != incidentID {
			continue
		}
		if status != "" && string(action.Status) != status {
			continue
		}
		result = append(result, action.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return capList(result, limit)
}

func (m *Memory) PutEvidence(evidence domain.Evidence) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.evidence[evidence.ID]; ok {
		return ErrExists
	}
	m.evidence[evidence.ID] = evidence.Clone()
	return nil
}

func (m *Memory) ListEvidence(incidentID string, limit int) []domain.Evidence {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Evidence, 0, len(m.evidence))
	for _, evidence := range m.evidence {
		if incidentID != "" && evidence.IncidentID != incidentID {
			continue
		}
		result = append(result, evidence.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return capList(result, limit)
}

func capList[T any](values []T, limit int) []T {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}
