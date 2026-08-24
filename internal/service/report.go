package service

import (
	"context"
	"sort"
	"time"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/store"
)

type OperationsReport struct {
	GeneratedAt       time.Time       `json:"generated_at"`
	Counts            store.Counts    `json:"counts"`
	OpenIncidents     int             `json:"open_incidents"`
	CriticalDevices   []domain.Device `json:"critical_devices"`
	TopSignals        []SignalSummary `json:"top_signals"`
	ActionSuccessRate float64         `json:"action_success_rate"`
}

type SignalSummary struct {
	Signal    string `json:"signal"`
	Anomalies int    `json:"anomalies"`
}

func (a *App) Report(ctx context.Context) (OperationsReport, error) {
	if err := a.ensureContext(ctx); err != nil {
		return OperationsReport{}, err
	}
	now := a.now()
	incidents := a.config.Repository.ListIncidents("", "", 0)
	devices := a.config.Repository.ListDevices("", 0)
	anomalies := a.config.Repository.ListAnomalies("", "", 0)
	actions := a.config.Repository.ListActions("", "", 0)
	report := OperationsReport{GeneratedAt: now, Counts: a.config.Repository.Counts(now)}
	for _, incident := range incidents {
		if incident.Status != domain.IncidentClosed && incident.Status != domain.IncidentResolved {
			report.OpenIncidents++
		}
	}
	for _, device := range devices {
		if device.Status == domain.DeviceDegraded {
			report.CriticalDevices = append(report.CriticalDevices, device)
		}
	}
	signalCounts := make(map[string]int)
	for _, anomaly := range anomalies {
		signalCounts[anomaly.Signal]++
	}
	for signal, count := range signalCounts {
		report.TopSignals = append(report.TopSignals, SignalSummary{Signal: signal, Anomalies: count})
	}
	sort.Slice(report.TopSignals, func(i, j int) bool {
		return report.TopSignals[i].Anomalies > report.TopSignals[j].Anomalies
	})
	if len(actions) > 0 {
		successful := 0
		for _, action := range actions {
			if action.Status == domain.ActionSucceeded {
				successful++
			}
		}
		report.ActionSuccessRate = float64(successful) / float64(len(actions))
	}
	return report, nil
}

func (a *App) Timeline(ctx context.Context, incidentID string) ([]map[string]any, error) {
	if _, err := a.GetIncident(ctx, incidentID); err != nil {
		return nil, err
	}
	evidence, err := a.ListEvidence(ctx, incidentID, 0)
	if err != nil {
		return nil, err
	}
	actions, err := a.ListActions(ctx, incidentID, "", 0)
	if err != nil {
		return nil, err
	}
	timeline := make([]map[string]any, 0, len(evidence)+len(actions))
	for _, item := range evidence {
		timeline = append(timeline, map[string]any{"kind": "evidence", "at": item.CreatedAt, "value": item})
	}
	for _, item := range actions {
		timeline = append(timeline, map[string]any{"kind": "action", "at": item.CreatedAt, "value": item})
	}
	sort.Slice(timeline, func(i, j int) bool {
		left := timeline[i]["at"].(time.Time)
		right := timeline[j]["at"].(time.Time)
		return left.Before(right)
	})
	return timeline, nil
}
