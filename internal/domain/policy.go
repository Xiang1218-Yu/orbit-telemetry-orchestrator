package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type PolicyStatus string

const (
	PolicyDraft     PolicyStatus = "draft"
	PolicyPublished PolicyStatus = "published"
	PolicyLocked    PolicyStatus = "locked"
	PolicyRetired   PolicyStatus = "retired"
)

type Comparator string

const (
	ComparatorAbove   Comparator = "above"
	ComparatorBelow   Comparator = "below"
	ComparatorOutside Comparator = "outside"
	ComparatorRate    Comparator = "rate"
)

type Rule struct {
	Signal     string        `json:"signal"`
	Comparator Comparator    `json:"comparator"`
	Lower      float64       `json:"lower"`
	Upper      float64       `json:"upper"`
	Duration   time.Duration `json:"duration"`
	Severity   int           `json:"severity"`
}

type ResponsePlan struct {
	ActionTypes []string      `json:"action_types"`
	MaxRetries  int           `json:"max_retries"`
	Cooldown    time.Duration `json:"cooldown"`
}

type CollectionPolicy struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	DeviceID       string       `json:"device_id"`
	Version        int          `json:"version"`
	Status         PolicyStatus `json:"status"`
	EffectiveFrom  time.Time    `json:"effective_from"`
	EffectiveUntil *time.Time   `json:"effective_until,omitempty"`
	Rules          []Rule       `json:"rules"`
	Response       ResponsePlan `json:"response"`
	CreatedBy      string       `json:"created_by"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

func NewPolicy(id, name, deviceID, creator string, version int, rules []Rule, response ResponsePlan, from time.Time, now time.Time) (CollectionPolicy, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(deviceID) == "" {
		return CollectionPolicy{}, errors.New("policy id, name, and device id are required")
	}
	if version < 1 {
		return CollectionPolicy{}, errors.New("policy version must be positive")
	}
	if len(rules) == 0 {
		return CollectionPolicy{}, errors.New("policy needs at least one rule")
	}
	for index := range rules {
		if err := rules[index].Validate(); err != nil {
			return CollectionPolicy{}, fmt.Errorf("rule %d: %w", index, err)
		}
	}
	if response.MaxRetries < 0 {
		return CollectionPolicy{}, errors.New("max retries cannot be negative")
	}
	return CollectionPolicy{
		ID: id, Name: strings.TrimSpace(name), DeviceID: deviceID, Version: version,
		Status: PolicyDraft, EffectiveFrom: from, Rules: append([]Rule(nil), rules...),
		Response: response, CreatedBy: creator, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (r Rule) Validate() error {
	if strings.TrimSpace(r.Signal) == "" {
		return errors.New("signal is required")
	}
	if r.Severity < 1 || r.Severity > 5 {
		return errors.New("severity must be between 1 and 5")
	}
	if r.Duration <= 0 {
		return errors.New("duration must be positive")
	}
	switch r.Comparator {
	case ComparatorAbove, ComparatorBelow:
		if r.Lower == 0 && r.Upper == 0 {
			return errors.New("threshold is required")
		}
	case ComparatorOutside:
		if r.Lower >= r.Upper {
			return errors.New("outside lower threshold must be below upper threshold")
		}
	case ComparatorRate:
		if r.Upper <= 0 {
			return errors.New("rate threshold must be positive")
		}
	default:
		return errors.New("unknown comparator")
	}
	return nil
}

func (p *CollectionPolicy) Publish(now time.Time) error {
	if p.Status != PolicyDraft {
		return errors.New("only draft policies can be published")
	}
	if p.EffectiveFrom.IsZero() {
		p.EffectiveFrom = now
	}
	p.Status = PolicyPublished
	p.UpdatedAt = now
	return nil
}

func (p *CollectionPolicy) Lock(now time.Time) error {
	if p.Status != PolicyPublished {
		return errors.New("only published policies can be locked")
	}
	p.Status = PolicyLocked
	p.UpdatedAt = now
	return nil
}

func (p *CollectionPolicy) Retire(now time.Time) error {
	if p.Status == PolicyRetired {
		return nil
	}
	p.Status = PolicyRetired
	p.UpdatedAt = now
	return nil
}

func (p CollectionPolicy) ActiveAt(now time.Time) bool {
	if p.Status != PolicyPublished && p.Status != PolicyLocked {
		return false
	}
	if now.Before(p.EffectiveFrom) {
		return false
	}
	return p.EffectiveUntil == nil || now.Before(*p.EffectiveUntil)
}

func (p CollectionPolicy) Clone() CollectionPolicy {
	p.Rules = append([]Rule(nil), p.Rules...)
	p.Response.ActionTypes = append([]string(nil), p.Response.ActionTypes...)
	if p.EffectiveUntil != nil {
		copy := *p.EffectiveUntil
		p.EffectiveUntil = &copy
	}
	return p
}
