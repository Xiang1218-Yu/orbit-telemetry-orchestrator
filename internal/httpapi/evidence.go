package httpapi

import (
	"net/http"

	"orbit-telemetry-orchestrator/internal/service"
)

func (s *Server) listEvidence(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	values, err := s.config.App.ListEvidence(r.Context(), id, serviceLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values, "count": len(values)})
}

func (s *Server) addEvidence(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input service.EvidenceInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.AddEvidence(r.Context(), actor(r), id, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
