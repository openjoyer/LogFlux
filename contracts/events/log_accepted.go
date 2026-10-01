package events

import "time"

type LogAccepted struct {
	EventID    string         `json:"event_id"`
	ProjectID  string         `json:"project_id"`
	Timestamp  time.Time      `json:"timestamp"`
	AcceptedAt time.Time      `json:"accepted_at"`
	Service    string         `json:"service"`
	Level      string         `json:"level"`
	Message    string         `json:"message"`
	Attributes map[string]any `json:"attributes"`
}
