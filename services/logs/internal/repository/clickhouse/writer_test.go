package clickhouse

import (
	"LogFlux/services/logs/internal/model"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type recordingConn struct {
	driver.Conn
	batch *recordingBatch
	query string
}

func (c *recordingConn) PrepareBatch(_ context.Context, query string, _ ...driver.PrepareBatchOption) (driver.Batch, error) {
	c.query = query
	return c.batch, nil
}

type recordingBatch struct {
	driver.Batch
	rows    [][]any
	sent    bool
	aborted bool
}

func (b *recordingBatch) Append(values ...any) error {
	b.rows = append(b.rows, append([]any(nil), values...))
	return nil
}

func (b *recordingBatch) Send() error {
	b.sent = true
	return nil
}

func (b *recordingBatch) Abort() error {
	b.aborted = true
	return nil
}

func (b *recordingBatch) IsSent() bool { return b.sent || b.aborted }

func TestWriterInsertsBatchWithExpectedValues(t *testing.T) {
	batch := &recordingBatch{}
	conn := &recordingConn{batch: batch}
	writer := NewWriter(conn)
	when := time.Date(2026, 10, 8, 20, 0, 0, 0, time.UTC)
	logs := []model.Log{
		{
			EventID:    "550e8400-e29b-41d4-a716-446655440000",
			ProjectID:  "project-1",
			Timestamp:  when,
			ReceivedAt: when.Add(time.Second),
			Service:    "orders",
			Level:      "ERROR",
			Message:    "timeout",
		},
		{
			EventID:     "550e8400-e29b-41d4-a716-446655440001",
			ProjectID:   "project-1",
			Timestamp:   when,
			ReceivedAt:  when.Add(time.Second),
			Service:     "billing",
			Environment: "production",
			Level:       "INFO",
			Message:     "paid",
			Attributes:  map[string]any{"order_id": "123"},
		},
	}

	if err := writer.InsertBatch(context.Background(), logs); err != nil {
		t.Fatal(err)
	}
	if !batch.sent || batch.aborted || len(batch.rows) != 2 {
		t.Fatalf("expected two sent rows, got sent=%t aborted=%t rows=%d", batch.sent, batch.aborted, len(batch.rows))
	}
	if !strings.Contains(conn.query, "INSERT INTO logflux.logs") {
		t.Fatalf("wrong insert target: %q", conn.query)
	}
	first := batch.rows[0]
	if len(first) != 9 || first[0] != "project-1" || first[5] != "unknown" || first[8] != "{}" ||
		first[2] != when || first[3] != when.Add(time.Second) {
		t.Fatalf("unexpected first row: %+v", first)
	}
	second := batch.rows[1]
	if second[5] != "production" {
		t.Fatalf("environment was lost: %+v", second)
	}
	var attributes map[string]any
	if err := json.Unmarshal([]byte(second[8].(string)), &attributes); err != nil {
		t.Fatal(err)
	}
	if attributes["order_id"] != "123" {
		t.Fatalf("attributes were changed: %+v", attributes)
	}
}

func TestWriterAbortsBatchOnInvalidEventID(t *testing.T) {
	batch := &recordingBatch{}
	writer := NewWriter(&recordingConn{batch: batch})
	logs := []model.Log{
		{EventID: "550e8400-e29b-41d4-a716-446655440000"},
		{EventID: "not-a-uuid"},
	}

	if err := writer.InsertBatch(context.Background(), logs); err == nil || !strings.Contains(err.Error(), "row 1: invalid event ID") {
		t.Fatalf("expected invalid event ID error, got %v", err)
	}
	if batch.sent || !batch.aborted {
		t.Fatalf("partial batch was not aborted: sent=%t aborted=%t", batch.sent, batch.aborted)
	}
}
