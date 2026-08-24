package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"orbit-telemetry-orchestrator/internal/service"
)

func TestBug003WrappedErrorsMissingStreamStatus(t *testing.T) {
	// source markers: POST /v1/telemetry service.IngestTelemetry writeAppError
	app := service.New(service.Config{})
	server := New(Config{App: app})
	body := `{"samples":[{"stream_id":"missing-stream","value":12,"quality":"good","observed_at":"2026-08-24T10:00:00Z","sequence":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", strings.NewReader(body)).WithContext(context.Background())
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing stream status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestBug003WrappedErrorsHealthRegression(t *testing.T) {
	// source markers: GET /healthz
	server := New(Config{App: service.New(service.Config{})})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", rec.Code, http.StatusOK)
	}
}
