package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"orbit-telemetry-orchestrator/internal/ratelimit"
)

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func recoverHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				_ = debug.Stack()
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func rateHandler(limiter *ratelimit.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("x-forwarded-for")
		if key == "" {
			key = r.RemoteAddr
		}
		if !limiter.Allow(key, time.Now().UTC()) {
			w.Header().Set("retry-after", "60")
			writeError(w, http.StatusTooManyRequests, errRateLimited{})
			return
		}
		next.ServeHTTP(w, r)
	})
}

type errRateLimited struct{}

func (errRateLimited) Error() string { return "request rate limit exceeded" }

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func writeAppError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, context.Canceled):
		// Client abandoned the request before the operation completed. The
		// service layer guarantees no system state was mutated, so report the
		// cancellation rather than framing it as a bad request.
		status = StatusClientClosedRequest
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
	case err.Error() == "record not found":
		status = http.StatusNotFound
	case err.Error() == "record already exists":
		status = http.StatusConflict
	case err.Error() == "record version conflict":
		status = http.StatusConflict
	}
	writeError(w, status, err)
}

// StatusClientClosedRequest indicates the client disconnected before the
// request finished. It mirrors nginx's 499 status used for cancelled requests.
const StatusClientClosedRequest = 499
