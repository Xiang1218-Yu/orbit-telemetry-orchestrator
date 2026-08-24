package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"orbit-telemetry-orchestrator/internal/service"
)

func TestBug007OptionalAuditMissingDependency(t *testing.T) {
	// source markers: httpapi.Server audit audit.Log.List GET /v1/audit
	server := New(Config{App: service.New(service.Config{}), Audit: nil})
	req := httptest.NewRequest(http.MethodGet, "/v1/audit", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestBug007OptionalAuditReadyRegression(t *testing.T) {
	// source markers: type Server GET /readyz
	server := New(Config{App: service.New(service.Config{}), Audit: nil})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want %d", rec.Code, http.StatusOK)
	}
}
