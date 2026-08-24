package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"orbit-telemetry-orchestrator/internal/audit"
	"orbit-telemetry-orchestrator/internal/clock"
	"orbit-telemetry-orchestrator/internal/events"
	"orbit-telemetry-orchestrator/internal/httpapi"
	"orbit-telemetry-orchestrator/internal/metrics"
	"orbit-telemetry-orchestrator/internal/queue"
	"orbit-telemetry-orchestrator/internal/ratelimit"
	"orbit-telemetry-orchestrator/internal/service"
	"orbit-telemetry-orchestrator/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	repository := store.NewMemory()
	bus := events.NewBus(events.Config{Buffer: 256, DropWhenFull: true})
	auditLog := audit.NewLog(5000)
	metricsRegistry := metrics.NewRegistry()
	limiter := ratelimit.New(120, time.Minute)
	workers := queue.New(queue.Config{Workers: 4, Buffer: 128, Logger: logger})
	app := service.New(service.Config{
		Repository: repository,
		Events:     bus,
		Audit:      auditLog,
		Queue:      workers,
		Metrics:    metricsRegistry,
		Clock:      clock.Real{},
		Logger:     logger,
	})
	workers.Register("incident-evaluation", app.ProcessIncidentEvaluation)
	workers.Register("response-action", app.ProcessResponseAction)
	workers.Register("evidence-rollup", app.ProcessEvidenceRollup)
	workers.Start()
	app.Seed()

	server := httpapi.New(httpapi.Config{
		App:     app,
		Events:  bus,
		Audit:   auditLog,
		Metrics: metricsRegistry,
		Limiter: limiter,
		Logger:  logger,
	})
	httpServer := &http.Server{
		Addr:              envOr("ORBIT_ADDR", ":8181"),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       45 * time.Second,
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = httpServer.Shutdown(shutdown)
		workers.Stop()
		bus.Close()
	}()
	logger.Info("orbit telemetry orchestrator listening", "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
