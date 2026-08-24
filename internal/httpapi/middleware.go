package httpapi

import (
	"encoding/json"
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
	status := appErrorStatus(err)
	writeError(w, status, err)
}

func appErrorStatus(err error) int {
	status := http.StatusBadRequest
	switch err.Error() {
	case "record not found":
		status = http.StatusNotFound
	case "record already exists":
		status = http.StatusConflict
	case "record version conflict":
		status = http.StatusConflict
	}
	return status
}
