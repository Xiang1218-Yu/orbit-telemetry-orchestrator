package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func serviceLimit(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	if parsed > 1000 {
		return 1000
	}
	return parsed
}

func pathID(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if id == "" {
		return "", errors.New("path id is required")
	}
	return id, nil
}

func actor(r *http.Request) string {
	value := r.Header.Get("x-orbit-actor")
	if value == "" {
		return "anonymous"
	}
	return value
}
