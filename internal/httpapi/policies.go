package httpapi

import (
	"net/http"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/service"
)

func (s *Server) listPolicies(w http.ResponseWriter, r *http.Request) {
	values, err := s.config.App.ListPolicies(
		r.Context(), r.URL.Query().Get("device_id"),
		domain.PolicyStatus(r.URL.Query().Get("status")),
		serviceLimit(r.URL.Query().Get("limit"), 100),
	)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values, "count": len(values)})
}

func (s *Server) createPolicy(w http.ResponseWriter, r *http.Request) {
	var input service.PolicyInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.CreatePolicy(r.Context(), actor(r), input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) getPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.GetPolicy(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) publishPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.PublishPolicy(r.Context(), actor(r), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) lockPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.LockPolicy(r.Context(), actor(r), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
