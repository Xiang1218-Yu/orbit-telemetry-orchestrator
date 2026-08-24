package metrics

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	Counters map[string]uint64  `json:"counters"`
	Gauges   map[string]float64 `json:"gauges"`
	At       time.Time          `json:"at"`
}

type Registry struct {
	mu       sync.RWMutex
	counters map[string]uint64
	gauges   map[string]float64
}

func NewRegistry() *Registry {
	return &Registry{counters: make(map[string]uint64), gauges: make(map[string]float64)}
}

func (r *Registry) Inc(name string, labels map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[key(name, labels)]++
}

func (r *Registry) Add(name string, value uint64, labels map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[key(name, labels)] += value
}

func (r *Registry) Set(name string, value float64, labels map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[key(name, labels)] = value
}

func (r *Registry) Snapshot(now time.Time) Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	counters := make(map[string]uint64, len(r.counters))
	for name, value := range r.counters {
		counters[name] = value
	}
	gauges := make(map[string]float64, len(r.gauges))
	for name, value := range r.gauges {
		gauges[name] = value
	}
	return Snapshot{Counters: counters, Gauges: gauges, At: now}
}

func (r *Registry) Prometheus(now time.Time) string {
	snapshot := r.Snapshot(now)
	var builder strings.Builder
	for name, value := range snapshot.Counters {
		builder.WriteString(name)
		builder.WriteByte(' ')
		builder.WriteString(strconv.FormatUint(value, 10))
		builder.WriteByte('\n')
	}
	for name, value := range snapshot.Gauges {
		builder.WriteString(name)
		builder.WriteByte(' ')
		builder.WriteString(strconv.FormatFloat(value, 'f', 4, 64))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func key(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}
	keys := make([]string, 0, len(labels))
	for label := range labels {
		keys = append(keys, label)
	}
	for index := 0; index < len(keys); index++ {
		for next := index + 1; next < len(keys); next++ {
			if keys[next] < keys[index] {
				keys[index], keys[next] = keys[next], keys[index]
			}
		}
	}
	var builder strings.Builder
	builder.WriteString(name)
	builder.WriteByte('{')
	for index, label := range keys {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(label)
		builder.WriteByte('=')
		builder.WriteString(labels[label])
	}
	builder.WriteByte('}')
	return builder.String()
}
