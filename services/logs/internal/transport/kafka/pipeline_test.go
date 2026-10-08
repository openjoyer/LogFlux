package kafka

import (
	"LogFlux/services/logs/internal/config"
	"LogFlux/services/logs/internal/model"
	"context"
	"errors"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type fakeConsumer struct {
	fetch  func(context.Context) (kafkago.Message, error)
	commit func(context.Context, ...kafkago.Message) error
}

func (f *fakeConsumer) Fetch(ctx context.Context) (kafkago.Message, error) {
	if f.fetch != nil {
		return f.fetch(ctx)
	}
	<-ctx.Done()
	return kafkago.Message{}, ctx.Err()
}

func (f *fakeConsumer) Commit(ctx context.Context, messages ...kafkago.Message) error {
	if f.commit != nil {
		return f.commit(ctx, messages...)
	}
	return nil
}

func (f *fakeConsumer) Close() error { return nil }

type fakeProcessor struct {
	process func(context.Context, []model.Log) error
}

func (f *fakeProcessor) Process(ctx context.Context, logs []model.Log) error {
	if f.process != nil {
		return f.process(ctx, logs)
	}
	return nil
}

func TestWriteWorkerCommitsOnlyAfterSuccessfulWrite(t *testing.T) {
	message := kafkago.Message{Partition: 2, Offset: 7}
	logs := []model.Log{{EventID: "event-1"}}
	var order []string

	consumer := &fakeConsumer{commit: func(_ context.Context, messages ...kafkago.Message) error {
		order = append(order, "commit")
		if len(messages) != 1 || messages[0].Partition != message.Partition || messages[0].Offset != message.Offset {
			t.Fatalf("committed wrong messages: %+v", messages)
		}
		return nil
	}}
	processor := &fakeProcessor{process: func(_ context.Context, received []model.Log) error {
		order = append(order, "write")
		if len(received) != 1 || received[0].EventID != logs[0].EventID {
			t.Fatalf("processed wrong logs: %+v", received)
		}
		return nil
	}}
	pipeline := NewPipeline(consumer, processor, config.BatchConfig{}, config.WriterConfig{
		InsertTimeout: time.Second,
		MaxAttempts:   1,
	})
	batches := make(chan Batch, 1)
	batches <- Batch{Logs: logs, Messages: []kafkago.Message{message}}
	close(batches)

	if err := pipeline.writeWorker(context.Background(), batches); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "write" || order[1] != "commit" {
		t.Fatalf("expected write before commit, got %v", order)
	}
}

func TestWriteWorkerDoesNotCommitFailedWrite(t *testing.T) {
	writeErr := errors.New("clickhouse unavailable")
	committed := false
	consumer := &fakeConsumer{commit: func(context.Context, ...kafkago.Message) error {
		committed = true
		return nil
	}}
	processor := &fakeProcessor{process: func(context.Context, []model.Log) error {
		return writeErr
	}}
	pipeline := NewPipeline(consumer, processor, config.BatchConfig{}, config.WriterConfig{
		InsertTimeout: time.Second,
		MaxAttempts:   1,
	})
	batches := make(chan Batch, 1)
	batches <- Batch{Logs: []model.Log{{}}, Messages: []kafkago.Message{{Offset: 7}}}
	close(batches)

	if err := pipeline.writeWorker(context.Background(), batches); !errors.Is(err, writeErr) {
		t.Fatalf("expected write error, got %v", err)
	}
	if committed {
		t.Fatal("committed a message before its batch was stored")
	}
}

func TestWriteWorkerReturnsCommitError(t *testing.T) {
	commitErr := errors.New("commit failed")
	consumer := &fakeConsumer{commit: func(context.Context, ...kafkago.Message) error {
		return commitErr
	}}
	pipeline := NewPipeline(consumer, &fakeProcessor{}, config.BatchConfig{}, config.WriterConfig{
		InsertTimeout: time.Second,
		MaxAttempts:   1,
	})
	batches := make(chan Batch, 1)
	batches <- Batch{Logs: []model.Log{{}}, Messages: []kafkago.Message{{Offset: 7}}}
	close(batches)

	if err := pipeline.writeWorker(context.Background(), batches); !errors.Is(err, commitErr) {
		t.Fatalf("expected commit error, got %v", err)
	}
}

func TestPipelineStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	consumer := &fakeConsumer{fetch: func(ctx context.Context) (kafkago.Message, error) {
		close(started)
		<-ctx.Done()
		return kafkago.Message{}, ctx.Err()
	}}
	pipeline := NewPipeline(consumer, &fakeProcessor{}, config.BatchConfig{}, config.WriterConfig{})
	result := make(chan error, 1)
	go func() { result <- pipeline.Run(ctx) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("pipeline did not start reading")
	}
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("pipeline did not stop after cancellation")
	}
}
