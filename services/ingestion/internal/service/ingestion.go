package service

import (
	"LogFlux/contracts/events"
	"LogFlux/services/ingestion/internal/transport/http"
	"context"
	"fmt"
	"strings"
	"time"
)

type LogsPublisher interface {
	PublishLogs(ctx context.Context, events []events.LogAccepted) error
}

type IngestionService struct {
	Publisher LogsPublisher
}

func NewIngestionService(p LogsPublisher) *IngestionService {
	return &IngestionService{p}
}

func (s *IngestionService) UploadLogs(ctx context.Context, logs []http.Log) error {
	logEvents := make([]events.LogAccepted, 0, len(logs))
	for i, log := range logs {
		event, err := mapToEvent(log)
		if err != nil {
			return fmt.Errorf("logs[%d]: %w", i, err)
		}

		logEvents = append(logEvents, event)
	}

	// TODO ???
	return s.Publisher.PublishLogs(ctx, logEvents)
}

func mapToEvent(log http.Log) (events.LogAccepted, error) {
	level, err := normalizeLevel(string(log.Level))
	if err != nil {
		return events.LogAccepted{}, err
	}

	return events.LogAccepted{
		EventID:    log.EventID,
		ProjectID:  "", // TODO check for projectID
		Timestamp:  log.Timestamp,
		AcceptedAt: time.Now(),
		Service:    log.Service,
		Level:      level,
		Message:    log.Message,
		Attributes: nil, // TODO add attributes
	}, nil
}

func normalizeLevel(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "TRACE":
		return "TRACE", nil
	case "DEBUG":
		return "DEBUG", nil
	case "INFO":
		return "INFO", nil
	case "WARN", "WARNING":
		return "WARN", nil
	case "ERROR", "ERR":
		return "ERROR", nil
	case "FATAL", "CRITICAL":
		return "FATAL", nil
	default:
		return "", fmt.Errorf("unsupported log level: %q", value)
	}
}
