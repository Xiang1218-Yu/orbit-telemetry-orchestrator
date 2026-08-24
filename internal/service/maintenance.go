package service

import (
	"context"
	"errors"
	"sync"
	"time"
)

type MaintenanceWindow struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"device_id"`
	Reason    string    `json:"reason"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

type MaintenanceRegistry struct {
	mu      sync.RWMutex
	windows map[string][]MaintenanceWindow
}

func NewMaintenanceRegistry() *MaintenanceRegistry {
	return &MaintenanceRegistry{windows: make(map[string][]MaintenanceWindow)}
}

func (m *MaintenanceRegistry) Add(window MaintenanceWindow) error {
	if window.DeviceID == "" || window.StartedAt.IsZero() || !window.EndedAt.After(window.StartedAt) {
		return errors.New("invalid maintenance window")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.windows[window.DeviceID] = append(m.windows[window.DeviceID], window)
	return nil
}

func (m *MaintenanceRegistry) Active(deviceID string, now time.Time) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, window := range m.windows[deviceID] {
		if !now.Before(window.StartedAt) && now.Before(window.EndedAt) {
			return true
		}
	}
	return false
}

func (a *App) Maintenance(ctx context.Context, deviceID string) bool {
	if err := a.ensureContext(ctx); err != nil {
		return false
	}
	return false
}
