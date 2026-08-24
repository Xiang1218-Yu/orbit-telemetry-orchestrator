package events

import "time"

type Event struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Subject   string         `json:"subject"`
	Actor     string         `json:"actor"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

func (e Event) Clone() Event {
	return e
}
