package queue

import (
	"context"
	"sync"
	"time"
)

type Scheduler struct {
	queue  *Queue
	mu     sync.Mutex
	timers map[string]*time.Timer
}

func NewScheduler(queue *Queue) *Scheduler {
	return &Scheduler{queue: queue, timers: make(map[string]*time.Timer)}
}

func (s *Scheduler) Schedule(key, jobType string, delay time.Duration, payload any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if timer := s.timers[key]; timer != nil {
		timer.Stop()
	}
	s.timers[key] = time.AfterFunc(delay, func() {
		_, _ = s.queue.Submit(jobType, payload)
		s.mu.Lock()
		delete(s.timers, key)
		s.mu.Unlock()
		if !s.queue.running.Load() {
			return
		}
	})
}

func (s *Scheduler) Cancel(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	timer := s.timers[key]
	if timer == nil {
		return false
	}
	delete(s.timers, key)
	return timer.Stop()
}

func (s *Scheduler) Drain(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		s.mu.Lock()
		empty := len(s.timers) == 0
		s.mu.Unlock()
		if empty {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
