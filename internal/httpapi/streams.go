package httpapi

import (
	"net/http"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/service"
)

func (s *Server) listStreams(w http.ResponseWriter, r *http.Request) {
	values, err := s.config.App.ListStreams(r.Context(), r.URL.Query().Get("device_id"), serviceLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values, "count": len(values)})
}

func (s *Server) createStream(w http.ResponseWriter, r *http.Request) {
	var input service.StreamInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.CreateStream(r.Context(), actor(r), input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) getStream(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.GetStream(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

type transitionStreamInput struct {
	Status domain.StreamStatus `json:"status"`
}

func (s *Server) transitionStream(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input transitionStreamInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.TransitionStream(r.Context(), actor(r), id, input.Status)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) listSamples(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	values, err := s.config.App.Samples(r.Context(), id, serviceLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values, "count": len(values)})
}
