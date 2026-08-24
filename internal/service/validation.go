package service

import (
	"errors"
	"net/http"
	"strings"
)

func requireActor(actor string) error {
	if strings.TrimSpace(actor) == "" {
		return errors.New("x-orbit-actor header is required")
	}
	return nil
}

func parseLimit(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	var result int
	for _, character := range value {
		if character < '0' || character > '9' {
			return fallback
		}
		result = result*10 + int(character-'0')
		if result > 1000 {
			return 1000
		}
	}
	if result == 0 {
		return fallback
	}
	return result
}

func actorFromRequest(request *http.Request) string {
	actor := strings.TrimSpace(request.Header.Get("x-orbit-actor"))
	if actor == "" {
		return "anonymous"
	}
	return actor
}
