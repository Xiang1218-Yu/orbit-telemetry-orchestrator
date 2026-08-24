package domain

import (
	"errors"
	"strings"
	"time"
)

type IncidentStatus string

const (
	IncidentOpen          IncidentStatus = "open"
	IncidentInvestigating IncidentStatus = "investigating"
	IncidentContained     IncidentStatus = "contained"
	IncidentResolved      IncidentStatus = "resolved"
	IncidentClosed        IncidentStatus = "closed"
)

type Incident struct {
	ID          string         `json:"id"`
	DeviceID    string         `json:"device_id"`
	Title       string         `json:"title"`
	Severity    int            `json:"severity"`
	Status      IncidentStatus `json:"status"`
	AnomalyIDs  []string       `json:"anomaly_ids"`
	ActionIDs   []string       `json:"action_ids"`
	EvidenceIDs []string       `json:"evidence_ids"`
	OpenedAt    time.Time      `json:"opened_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	ClosedAt    *time.Time     `json:"closed_at,omitempty"`
	Version     int64          `json:"version"`
}

func NewIncident(id, deviceID, title string, severity int, anomalyID string, now time.Time) (Incident, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(deviceID) == "" || strings.TrimSpace(title) == "" {
		return Incident{}, errors.New("incident id, device id, and title are required")
	}
	if severity < 1 || severity > 5 {
		return Incident{}, errors.New("incident severity must be between 1 and 5")
	}
	return Incident{
		ID: id, DeviceID: deviceID, Title: strings.TrimSpace(title), Severity: severity,
		Status: IncidentOpen, AnomalyIDs: []string{anomalyID}, OpenedAt: now,
		UpdatedAt: now, Version: 1,
	}, nil
}

func (i Incident) Clone() Incident {
	i.AnomalyIDs = append([]string(nil), i.AnomalyIDs...)
	i.ActionIDs = append([]string(nil), i.ActionIDs...)
	i.EvidenceIDs = append([]string(nil), i.EvidenceIDs...)
	if i.ResolvedAt != nil {
		copy := *i.ResolvedAt
		i.ResolvedAt = &copy
	}
	if i.ClosedAt != nil {
		copy := *i.ClosedAt
		i.ClosedAt = &copy
	}
	return i
}

func (i *Incident) AddAnomaly(anomalyID string, severity int, now time.Time) error {
	if i.Status == IncidentClosed {
		return errors.New("closed incident cannot receive anomalies")
	}
	for _, existing := range i.AnomalyIDs {
		if existing == anomalyID {
			return nil
		}
	}
	i.AnomalyIDs = append(i.AnomalyIDs, anomalyID)
	if severity > i.Severity {
		i.Severity = severity
	}
	i.Version++
	i.UpdatedAt = now
	return nil
}

func (i *Incident) Transition(status IncidentStatus, now time.Time) error {
	if status == i.Status {
		return nil
	}
	allowed := map[IncidentStatus][]IncidentStatus{
		IncidentOpen:          {IncidentInvestigating, IncidentContained, IncidentResolved},
		IncidentInvestigating: {IncidentContained, IncidentResolved},
		IncidentContained:     {IncidentInvestigating, IncidentResolved},
		IncidentResolved:      {IncidentClosed, IncidentInvestigating},
		IncidentClosed:        {},
	}
	next, ok := allowed[i.Status]
	if !ok {
		return errors.New("unknown incident status")
	}
	for _, option := range next {
		if option == status {
			i.Status = status
			i.Version++
			i.UpdatedAt = now
			if status == IncidentResolved {
				copy := now
				i.ResolvedAt = &copy
			}
			if status == IncidentClosed {
				copy := now
				i.ClosedAt = &copy
			}
			return nil
		}
	}
	return errors.New("invalid incident transition")
}

func (i *Incident) AttachAction(actionID string, now time.Time) error {
	if i.Status == IncidentClosed {
		return errors.New("closed incident cannot receive actions")
	}
	i.ActionIDs = appendUnique(i.ActionIDs, actionID)
	i.Version++
	i.UpdatedAt = now
	return nil
}

func (i *Incident) AttachEvidence(evidenceID string, now time.Time) error {
	if i.Status == IncidentClosed {
		return errors.New("closed incident cannot receive evidence")
	}
	i.EvidenceIDs = appendUnique(i.EvidenceIDs, evidenceID)
	i.Version++
	i.UpdatedAt = now
	return nil
}

func (i Incident) ReadyToClose() bool {
	return i.Status == IncidentResolved && len(i.EvidenceIDs) > 0
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
