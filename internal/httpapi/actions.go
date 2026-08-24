package httpapi

import (
	"net/http"

	"orbit-telemetry-orchestrator/internal/service"
)

func (s *Server) listActions(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	values, err := s.config.App.ListActions(r.Context(), id, r.URL.Query().Get("status"), serviceLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values, "count": len(values)})
}

func (s *Server) createAction(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input service.ActionInput
	if !decode(w, r, &input) {
		return
	}
	input.IncidentID = id
	value, err := s.config.App.CreateAction(r.Context(), actor(r), input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, value)
}

func (s *Server) getAction(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.GetAction(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) executeAction(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.ExecuteAction(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) cancelAction(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.CancelAction(r.Context(), actor(r), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
