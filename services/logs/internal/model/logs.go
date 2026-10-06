package model

import "time"

type Log struct {
	EventID     string
	ProjectID   string
	Timestamp   time.Time
	ReceivedAt  time.Time
	Service     string
	Level       string
	Message     string
	Environment string
	Attributes  map[string]any
}
