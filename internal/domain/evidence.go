package domain

import (
	"errors"
	"strings"
	"time"
)

type EvidenceKind string

const (
	EvidenceTelemetry EvidenceKind = "telemetry-window"
	EvidenceAction    EvidenceKind = "action-result"
	EvidenceNote      EvidenceKind = "operator-note"
	EvidenceSnapshot  EvidenceKind = "device-snapshot"
)

type Evidence struct {
	ID          string       `json:"id"`
	IncidentID  string       `json:"incident_id"`
	Kind        EvidenceKind `json:"kind"`
	Title       string       `json:"title"`
	Body        string       `json:"body"`
	Source      string       `json:"source"`
	CollectedAt time.Time    `json:"collected_at"`
	CreatedAt   time.Time    `json:"created_at"`
}

func NewEvidence(id, incidentID string, kind EvidenceKind, title, body, source string, now time.Time) (Evidence, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(incidentID) == "" {
		return Evidence{}, errors.New("evidence id and incident id are required")
	}
	if strings.TrimSpace(title) == "" || strings.TrimSpace(body) == "" {
		return Evidence{}, errors.New("evidence title and body are required")
	}
	switch kind {
	case EvidenceTelemetry, EvidenceAction, EvidenceNote, EvidenceSnapshot:
	default:
		return Evidence{}, errors.New("unknown evidence kind")
	}
	return Evidence{
		ID: id, IncidentID: incidentID, Kind: kind, Title: strings.TrimSpace(title),
		Body: body, Source: strings.TrimSpace(source), CollectedAt: now, CreatedAt: now,
	}, nil
}

func (e Evidence) Clone() Evidence {
	return e
}
