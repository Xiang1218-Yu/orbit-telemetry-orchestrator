package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"orbit-telemetry-orchestrator/internal/audit"
	"orbit-telemetry-orchestrator/internal/events"
	"orbit-telemetry-orchestrator/internal/metrics"
	"orbit-telemetry-orchestrator/internal/ratelimit"
	"orbit-telemetry-orchestrator/internal/service"
)

type Config struct {
	App     *service.App
	Events  *events.Bus
	Audit   *audit.Log
	Metrics *metrics.Registry
	Limiter *ratelimit.Limiter
	Logger  *slog.Logger
}

type Server struct {
	config Config
	mux    *http.ServeMux
}

func New(config Config) *Server {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	server := &Server{config: config, mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.mux
	handler = recoverHandler(handler)
	handler = requestLogger(s.config.Logger, handler)
	if s.config.Limiter != nil {
		handler = rateHandler(s.config.Limiter, handler)
	}
	return handler
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /readyz", s.ready)
	s.mux.HandleFunc("GET /metrics", s.metrics)
	s.mux.HandleFunc("GET /v1/stats", s.stats)
	s.mux.HandleFunc("GET /v1/report", s.report)
	s.mux.HandleFunc("GET /v1/audit", s.audit)
	s.mux.HandleFunc("GET /v1/events", s.eventsStream)

	s.mux.HandleFunc("GET /v1/devices", s.listDevices)
	s.mux.HandleFunc("POST /v1/devices", s.createDevice)
	s.mux.HandleFunc("GET /v1/devices/{id}", s.getDevice)
	s.mux.HandleFunc("POST /v1/devices/{id}/transition", s.transitionDevice)

	s.mux.HandleFunc("GET /v1/streams", s.listStreams)
	s.mux.HandleFunc("POST /v1/streams", s.createStream)
	s.mux.HandleFunc("GET /v1/streams/{id}", s.getStream)
	s.mux.HandleFunc("POST /v1/streams/{id}/transition", s.transitionStream)
	s.mux.HandleFunc("GET /v1/streams/{id}/samples", s.listSamples)

	s.mux.HandleFunc("GET /v1/policies", s.listPolicies)
	s.mux.HandleFunc("POST /v1/policies", s.createPolicy)
	s.mux.HandleFunc("GET /v1/policies/{id}", s.getPolicy)
	s.mux.HandleFunc("POST /v1/policies/{id}/publish", s.publishPolicy)
	s.mux.HandleFunc("POST /v1/policies/{id}/lock", s.lockPolicy)

	s.mux.HandleFunc("POST /v1/telemetry", s.ingestTelemetry)
	s.mux.HandleFunc("GET /v1/anomalies", s.listAnomalies)
	s.mux.HandleFunc("GET /v1/incidents", s.listIncidents)
	s.mux.HandleFunc("GET /v1/incidents/{id}", s.getIncident)
	s.mux.HandleFunc("POST /v1/incidents/{id}/transition", s.transitionIncident)
	s.mux.HandleFunc("GET /v1/incidents/{id}/actions", s.listActions)
	s.mux.HandleFunc("POST /v1/incidents/{id}/actions", s.createAction)
	s.mux.HandleFunc("GET /v1/incidents/{id}/evidence", s.listEvidence)
	s.mux.HandleFunc("POST /v1/incidents/{id}/evidence", s.addEvidence)
	s.mux.HandleFunc("GET /v1/incidents/{id}/timeline", s.timeline)
	s.mux.HandleFunc("GET /v1/actions/{id}", s.getAction)
	s.mux.HandleFunc("POST /v1/actions/{id}/execute", s.executeAction)
	s.mux.HandleFunc("POST /v1/actions/{id}/cancel", s.cancelAction)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "orbit", "time": time.Now().UTC()})
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	if s.config.Metrics == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("metrics are not configured"))
		return
	}
	w.Header().Set("content-type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(s.config.Metrics.Prometheus(time.Now().UTC())))
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if s.config.App == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("application is not configured"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ready": true})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	result, err := s.config.App.Stats(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	result, err := s.config.App.Report(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	result := s.config.Audit.List(audit.Filter{
		Actor: r.URL.Query().Get("actor"), Action: r.URL.Query().Get("action"),
		Resource: r.URL.Query().Get("resource"),
		Limit:    serviceLimit(r.URL.Query().Get("limit"), 100),
	})
	writeJSON(w, http.StatusOK, map[string]any{"items": result, "count": len(result)})
}

func (s *Server) eventsStream(w http.ResponseWriter, r *http.Request) {
	if s.config.Events == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("event bus is not configured"))
		return
	}
	subscription := s.config.Events.Subscribe(r.URL.Query().Get("topic"))
	defer subscription.Close()
	w.Header().Set("content-type", "application/x-ndjson")
	flusher, _ := w.(http.Flusher)
	encoder := json.NewEncoder(w)
	for {
		select {
		case <-r.Context().Done():
			return
			case event, ok := <-subscription.Events():
				if !ok {
					return
				}
				event = event.Clone()
				if err := encoder.Encode(event); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}
