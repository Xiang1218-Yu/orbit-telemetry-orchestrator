package httpapi

import (
	"net/http"

	"orbit-telemetry-orchestrator/internal/service"
)

func (s *Server) ingestTelemetry(w http.ResponseWriter, r *http.Request) {
	var input service.TelemetryBatch
	if !decode(w, r, &input) {
		return
	}
	count, err := s.config.App.IngestTelemetry(r.Context(), actor(r), input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": count})
}

func (s *Server) listAnomalies(w http.ResponseWriter, r *http.Request) {
	values, err := s.config.App.ListAnomalies(
		r.Context(), r.URL.Query().Get("device_id"), r.URL.Query().Get("status"),
		serviceLimit(r.URL.Query().Get("limit"), 100),
	)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values, "count": len(values)})
}
