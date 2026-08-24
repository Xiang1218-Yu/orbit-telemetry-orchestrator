package queue

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

var ErrStopped = errors.New("queue stopped")
var ErrFull = errors.New("queue full")

type Job struct {
	ID      string
	Type    string
	Payload any
	Created time.Time
}

type Handler func(context.Context, Job) error

type Config struct {
	Workers int
	Buffer  int
	Logger  *slog.Logger
}

type Queue struct {
	config   Config
	jobs     chan Job
	handlers map[string]Handler
	mu       sync.RWMutex
	wg       sync.WaitGroup
	nextID   uint64
	ctx      context.Context
	cancel   context.CancelFunc
	running  atomic.Bool
}

func New(config Config) *Queue {
	if config.Workers < 1 {
		config.Workers = 1
	}
	if config.Buffer < 1 {
		config.Buffer = 16
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Queue{
		config: config, jobs: make(chan Job, config.Buffer),
		handlers: make(map[string]Handler), ctx: ctx, cancel: cancel,
	}
}

func (q *Queue) Register(name string, handler Handler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[name] = handler
}

func (q *Queue) Start() {
	if q.running.Swap(true) {
		return
	}
	for index := 0; index < q.config.Workers; index++ {
		q.wg.Add(1)
		go q.worker(index)
	}
}

func (q *Queue) Submit(jobType string, payload any) (string, error) {
	if !q.running.Load() {
		return "", ErrStopped
	}
	job := Job{
		ID: "job-" + itoa(atomic.AddUint64(&q.nextID, 1)), Type: jobType,
		Payload: payload, Created: time.Now().UTC(),
	}
	select {
	case q.jobs <- job:
		return job.ID, nil
	default:
		return "", ErrFull
	}
}

func (q *Queue) Stop() {
	if !q.running.Swap(false) {
		return
	}
	q.cancel()
	q.wg.Wait()
}

func (q *Queue) worker(index int) {
	defer q.wg.Done()
	for {
		select {
		case <-q.ctx.Done():
			return
		case job := <-q.jobs:
			q.handle(index, job)
		}
	}
}

func (q *Queue) handle(index int, job Job) {
	q.mu.RLock()
	handler := q.handlers[job.Type]
	q.mu.RUnlock()
	if handler == nil {
		q.config.Logger.Error("queue handler missing", "worker", index, "job", job.ID, "type", job.Type)
		return
	}
	if err := handler(q.ctx, job); err != nil {
		q.config.Logger.Error("queue job failed", "worker", index, "job", job.ID, "type", job.Type, "error", err)
	}
}

func itoa(value uint64) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[index:])
}
