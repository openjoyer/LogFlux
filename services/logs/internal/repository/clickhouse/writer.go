package clickhouse

import (
	"LogFlux/services/logs/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Writer struct {
	conn driver.Conn
}

func NewWriter(conn driver.Conn) *Writer {
	return &Writer{conn: conn}
}

func (w *Writer) InsertBatch(ctx context.Context, logs []model.Log) error {
	if len(logs) == 0 {
		return nil
	}

	batch, err := w.conn.PrepareBatch(ctx, `
		INSERT INTO logflux.logs (
			project_id,
			event_id,
			occurred_at,
			received_at,
			service,
			environment,
			level,
			message,
			attributes
		)
	`)

	if err != nil {
		return fmt.Errorf("prepare logs batch: %w", err)
	}

	defer func() {
		if !batch.IsSent() {
			_ = batch.Abort()
		}
	}()

	for i, entry := range logs {
		eventID, err := uuid.Parse(entry.EventID)
		if err != nil {
			return fmt.Errorf("row %d: invalid event ID: %w", i, err)
		}

		attributes := "{}"
		if entry.Attributes != nil {
			data, err := json.Marshal(entry.Attributes)
			if err != nil {
				return fmt.Errorf("row %d: encode attributes: %w", i, err)
			}
			attributes = string(data)
		}

		environment := entry.Environment
		if environment == "" {
			environment = "unknown"
		}

		if err := batch.Append(
			entry.ProjectID,
			eventID,
			entry.Timestamp,
			entry.ReceivedAt,
			entry.Service,
			environment,
			entry.Level,
			entry.Message,
			attributes,
		); err != nil {
			return fmt.Errorf("append row %d: %w", i, err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send logs batch: %w", err)
	}

	return nil
}
