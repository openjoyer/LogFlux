package kafka

import (
	"LogFlux/contracts/events"
	"LogFlux/services/logs/internal/config"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func TestReadBatchMapsEventsAndStopsAtEventLimit(t *testing.T) {
	acceptedAt := time.Date(2026, 10, 8, 20, 0, 0, 0, time.UTC)
	event := events.LogAccepted{
		EventID:    "event-1",
		ProjectID:  "project-1",
		Timestamp:  acceptedAt.Add(-time.Second),
		AcceptedAt: acceptedAt,
		Service:    "orders",
		Level:      "ERROR",
		Message:    "timeout",
		Attributes: map[string]any{"region": "eu"},
	}
	consumer := queuedConsumer(t, []kafkago.Message{
		acceptedMessage(t, event, 0),
		acceptedMessage(t, event, 1),
	})
	pipeline := NewPipeline(consumer, nil, config.BatchConfig{
		MaxEvents:     2,
		MaxBytes:      1 << 20,
		FlushInterval: time.Second,
	}, config.WriterConfig{})

	batch, err := pipeline.readBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Logs) != 2 || len(batch.Messages) != 2 {
		t.Fatalf("expected two logs and messages, got %+v", batch)
	}
	got := batch.Logs[0]
	if got.EventID != event.EventID || got.ProjectID != event.ProjectID || got.Service != event.Service ||
		got.Level != event.Level || got.Message != event.Message || !got.Timestamp.Equal(event.Timestamp) ||
		!got.ReceivedAt.Equal(event.AcceptedAt) || got.Attributes["region"] != "eu" {
		t.Fatalf("event mapping is incomplete: %+v", got)
	}
	if batch.Messages[0].Offset != 0 || batch.Messages[1].Offset != 1 {
		t.Fatalf("message offsets changed: %+v", batch.Messages)
	}
}

func TestReadBatchStopsAtByteLimit(t *testing.T) {
	message := acceptedMessage(t, events.LogAccepted{EventID: "event-1"}, 4)
	message.Key = []byte("project-1")
	consumer := queuedConsumer(t, []kafkago.Message{message})
	pipeline := NewPipeline(consumer, nil, config.BatchConfig{
		MaxEvents:     10,
		MaxBytes:      len(message.Key) + len(message.Value),
		FlushInterval: time.Second,
	}, config.WriterConfig{})

	batch, err := pipeline.readBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Messages) != 1 || batch.Messages[0].Offset != 4 {
		t.Fatalf("expected one message at byte limit, got %+v", batch.Messages)
	}
}

func TestReadBatchFlushesAfterFirstMessageTimesOut(t *testing.T) {
	consumer := queuedConsumer(t, []kafkago.Message{
		acceptedMessage(t, events.LogAccepted{EventID: "event-1"}, 3),
	})
	pipeline := NewPipeline(consumer, nil, config.BatchConfig{
		MaxEvents:     10,
		MaxBytes:      1 << 20,
		FlushInterval: 20 * time.Millisecond,
	}, config.WriterConfig{})

	batch, err := pipeline.readBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Messages) != 1 || batch.Messages[0].Offset != 3 {
		t.Fatalf("expected pending message to flush on timeout, got %+v", batch.Messages)
	}
}

func TestReadBatchRejectsMalformedMessage(t *testing.T) {
	consumer := queuedConsumer(t, []kafkago.Message{{
		Partition: 2,
		Offset:    7,
		Value:     []byte("{"),
	}})
	pipeline := NewPipeline(consumer, nil, config.BatchConfig{FlushInterval: time.Second}, config.WriterConfig{})

	batch, err := pipeline.readBatch(context.Background())
	if err == nil || !strings.Contains(err.Error(), "partition=2 offset=7") {
		t.Fatalf("expected error with Kafka position, got %v", err)
	}
	if len(batch.Messages) != 0 {
		t.Fatalf("malformed message should not produce a batch: %+v", batch)
	}
}

func acceptedMessage(t *testing.T, event events.LogAccepted, offset int64) kafkago.Message {
	t.Helper()
	value, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return kafkago.Message{Partition: 1, Offset: offset, Value: value}
}

func queuedConsumer(t *testing.T, messages []kafkago.Message) *fakeConsumer {
	t.Helper()
	return &fakeConsumer{fetch: func(ctx context.Context) (kafkago.Message, error) {
		if len(messages) > 0 {
			message := messages[0]
			messages = messages[1:]
			return message, nil
		}
		<-ctx.Done()
		return kafkago.Message{}, ctx.Err()
	}}
}
