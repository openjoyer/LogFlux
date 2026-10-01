package http

import "time"

type Level string

const (
	LevelTrace Level = "trace"
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelFatal Level = "fatal"
)

type Log struct {
	EventID   string    `json:"event_id"`
	Message   string    `json:"message"`
	Level     Level     `json:"level"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}
