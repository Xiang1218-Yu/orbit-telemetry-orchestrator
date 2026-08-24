package httpapi

import (
	"net/http"

	"orbit-telemetry-orchestrator/internal/domain"
	"orbit-telemetry-orchestrator/internal/service"
)

func (s *Server) listDevices(w http.ResponseWriter, r *http.Request) {
	values, err := s.config.App.ListDevices(r.Context(), domain.DeviceStatus(r.URL.Query().Get("status")), serviceLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values, "count": len(values)})
}

func (s *Server) createDevice(w http.ResponseWriter, r *http.Request) {
	var input service.DeviceInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.CreateDevice(r.Context(), actor(r), input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) getDevice(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.config.App.GetDevice(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

type transitionDeviceInput struct {
	Status domain.DeviceStatus `json:"status"`
}

func (s *Server) transitionDevice(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var input transitionDeviceInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.TransitionDevice(r.Context(), actor(r), id, input.Status)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
