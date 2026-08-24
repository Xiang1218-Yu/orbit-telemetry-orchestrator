package domain

import (
	"errors"
	"strings"
	"time"
)

type DeviceStatus string

const (
	DevicePending  DeviceStatus = "pending"
	DeviceReady    DeviceStatus = "ready"
	DeviceDegraded DeviceStatus = "degraded"
	DeviceRetired  DeviceStatus = "retired"
)

type Device struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Site         string       `json:"site"`
	Model        string       `json:"model"`
	Status       DeviceStatus `json:"status"`
	Labels       []string     `json:"labels"`
	LastSeenAt   *time.Time   `json:"last_seen_at,omitempty"`
	FailureCount int          `json:"failure_count"`
	Version      int64        `json:"version"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

func NewDevice(id, name, site, model string, labels []string, now time.Time) (Device, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(name) == "" {
		return Device{}, errors.New("device id and name are required")
	}
	if strings.TrimSpace(site) == "" || strings.TrimSpace(model) == "" {
		return Device{}, errors.New("device site and model are required")
	}
	return Device{
		ID: id, Name: strings.TrimSpace(name), Site: strings.TrimSpace(site),
		Model: strings.TrimSpace(model), Status: DevicePending,
		Labels: NormalizeLabels(labels), Version: 1, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (d Device) CanReceiveTelemetry() bool {
	return d.Status == DeviceReady || d.Status == DeviceDegraded
}

func (d Device) Clone() Device {
	d.Labels = append([]string(nil), d.Labels...)
	if d.LastSeenAt != nil {
		copy := *d.LastSeenAt
		d.LastSeenAt = &copy
	}
	return d
}

func (d *Device) Transition(status DeviceStatus, now time.Time) error {
	if status == d.Status {
		return nil
	}
	switch d.Status {
	case DevicePending:
		if status != DeviceReady && status != DeviceRetired {
			return errors.New("pending device can only become ready or retired")
		}
	case DeviceReady:
		if status != DeviceDegraded && status != DeviceRetired {
			return errors.New("ready device can only become degraded or retired")
		}
	case DeviceDegraded:
		if status != DeviceReady && status != DeviceRetired {
			return errors.New("degraded device can only become ready or retired")
		}
	case DeviceRetired:
		return errors.New("retired device cannot transition")
	default:
		return errors.New("unknown device status")
	}
	d.Status = status
	d.Version++
	d.UpdatedAt = now
	return nil
}

func NormalizeLabels(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
