package httpapi

import (
	"net/http"

	"orbit-telemetry-orchestrator/internal/domain"
)

func (s *Server) listIncidents(w http.ResponseWriter, r *http.Request) {
	var values []domain.Incident
	var err error
	if query := r.URL.Query().Get("q"); query != "" {
		values, err = s.config.App.SearchIncidents(r.Context(), query, serviceLimit(r.URL.Query().Get("limit"), 100))
	} else {
		values, err = s.config.App.ListIncidents(
			r.Context(), r.URL.Query().Get("device_id"), r.URL.Query().Get("status"),
			serviceLimit(r.URL.Query().Get("limit"), 100),
		)
	}
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values, "count": len(values)})
}

func (s *Server) getIncident(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.GetIncident(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

type transitionIncidentInput struct {
	Status domain.IncidentStatus `json:"status"`
}

func (s *Server) transitionIncident(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input transitionIncidentInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.TransitionIncident(r.Context(), actor(r), id, input.Status)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) timeline(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.Timeline(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": value, "count": len(value)})
}
