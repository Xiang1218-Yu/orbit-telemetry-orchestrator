package domain

import (
	"errors"
	"time"
)

type AnomalyStatus string

const (
	AnomalyOpen         AnomalyStatus = "open"
	AnomalyAcknowledged AnomalyStatus = "acknowledged"
	AnomalyResolved     AnomalyStatus = "resolved"
)

type Anomaly struct {
	ID         string        `json:"id"`
	DeviceID   string        `json:"device_id"`
	StreamID   string        `json:"stream_id"`
	Signal     string        `json:"signal"`
	PolicyID   string        `json:"policy_id"`
	RuleIndex  int           `json:"rule_index"`
	Severity   int           `json:"severity"`
	Observed   float64       `json:"observed"`
	Threshold  string        `json:"threshold"`
	Window     Window        `json:"window"`
	Status     AnomalyStatus `json:"status"`
	IncidentID string        `json:"incident_id,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

func (a Anomaly) Clone() Anomaly {
	return a
}

func (a *Anomaly) Acknowledge(now time.Time) error {
	if a.Status != AnomalyOpen {
		return errors.New("only open anomalies can be acknowledged")
	}
	a.Status = AnomalyAcknowledged
	a.UpdatedAt = now
	return nil
}

func (a *Anomaly) Resolve(now time.Time) error {
	if a.Status == AnomalyResolved {
		return nil
	}
	a.Status = AnomalyResolved
	a.UpdatedAt = now
	return nil
}
