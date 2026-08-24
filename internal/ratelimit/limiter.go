package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	start time.Time
	count int
}

type Limiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	values map[string]bucket
}

func New(limit int, window time.Duration) *Limiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &Limiter{limit: limit, window: window, values: make(map[string]bucket)}
}

func (l *Limiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	current := l.values[key]
	if current.start.IsZero() || !now.Before(current.start.Add(l.window)) {
		current = bucket{start: now, count: 0}
	}
	if current.count >= l.limit {
		l.values[key] = current
		return false
	}
	current.count++
	l.values[key] = current
	return true
}

func (l *Limiter) Remaining(key string, now time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	current := l.values[key]
	if current.start.IsZero() || !now.Before(current.start.Add(l.window)) {
		return l.limit
	}
	remaining := l.limit - current.count
	if remaining < 0 {
		return 0
	}
	return remaining
}
