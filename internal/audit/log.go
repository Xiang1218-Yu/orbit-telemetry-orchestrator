package audit

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Entry struct {
	ID         string            `json:"id"`
	Actor      string            `json:"actor"`
	Action     string            `json:"action"`
	Resource   string            `json:"resource"`
	ResourceID string            `json:"resource_id"`
	Message    string            `json:"message"`
	Changes    map[string]string `json:"changes,omitempty"`
	OccurredAt time.Time         `json:"occurred_at"`
}

type Filter struct {
	Actor    string
	Action   string
	Resource string
	Limit    int
}

type Log struct {
	mu       sync.RWMutex
	limit    int
	sequence uint64
	entries  []Entry
}

func NewLog(limit int) *Log {
	if limit < 1 {
		limit = 1000
	}
	return &Log{limit: limit, entries: make([]Entry, 0, limit)}
}

func (l *Log) Record(actor, action, resource, resourceID, message string, changes map[string]string, now time.Time) Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sequence++
	entry := Entry{
		ID:    fmt.Sprintf("audit-%d-%d", now.UnixNano(), l.sequence),
		Actor: actor, Action: action, Resource: resource, ResourceID: resourceID,
		Message: message, Changes: cloneMap(changes), OccurredAt: now,
	}
	l.entries = append(l.entries, entry)
	if len(l.entries) > l.limit {
		l.entries = append([]Entry(nil), l.entries[len(l.entries)-l.limit:]...)
	}
	return clone(entry)
}

func (l *Log) List(filter Filter) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]Entry, 0, len(l.entries))
	for _, entry := range l.entries {
		if filter.Actor != "" && filter.Actor != entry.Actor {
			continue
		}
		if filter.Action != "" && filter.Action != entry.Action {
			continue
		}
		if filter.Resource != "" && filter.Resource != entry.Resource {
			continue
		}
		result = append(result, clone(entry))
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].OccurredAt.Equal(result[j].OccurredAt) {
			return result[i].ID > result[j].ID
		}
		return result[i].OccurredAt.After(result[j].OccurredAt)
	})
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	return result
}

func (l *Log) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

func clone(entry Entry) Entry {
	entry.Changes = cloneMap(entry.Changes)
	return entry
}

func cloneMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	target := make(map[string]string, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}
