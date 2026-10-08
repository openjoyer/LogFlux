package kafka

import (
	"LogFlux/services/logs/internal/config"
	"LogFlux/services/logs/internal/model"
	"context"
	"errors"
	"testing"
	"time"
)

func TestInsertWithRetryRetriesTemporaryFailure(t *testing.T) {
	calls := 0
	processor := &fakeProcessor{process: func(context.Context, []model.Log) error {
		calls++
		if calls == 1 {
			return context.DeadlineExceeded
		}
		return nil
	}}
	pipeline := NewPipeline(nil, processor, config.BatchConfig{}, retryConfig(3))

	if err := pipeline.insertWithRetry(context.Background(), retryConfig(3), []model.Log{{}}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected one retry, got %d attempts", calls)
	}
}

func TestInsertWithRetryStopsOnPermanentFailure(t *testing.T) {
	writeErr := errors.New("invalid row")
	calls := 0
	processor := &fakeProcessor{process: func(context.Context, []model.Log) error {
		calls++
		return writeErr
	}}
	pipeline := NewPipeline(nil, processor, config.BatchConfig{}, retryConfig(3))

	if err := pipeline.insertWithRetry(context.Background(), retryConfig(3), nil); !errors.Is(err, writeErr) {
		t.Fatalf("expected permanent write error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("permanent error was retried %d times", calls)
	}
}

func TestInsertWithRetryStopsAfterMaximumAttempts(t *testing.T) {
	calls := 0
	processor := &fakeProcessor{process: func(context.Context, []model.Log) error {
		calls++
		return context.DeadlineExceeded
	}}
	pipeline := NewPipeline(nil, processor, config.BatchConfig{}, retryConfig(2))

	if err := pipeline.insertWithRetry(context.Background(), retryConfig(2), nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected last write error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected two attempts, got %d", calls)
	}
}

func TestInsertWithRetryRejectsZeroAttempts(t *testing.T) {
	called := false
	processor := &fakeProcessor{process: func(context.Context, []model.Log) error {
		called = true
		return nil
	}}
	pipeline := NewPipeline(nil, processor, config.BatchConfig{}, config.WriterConfig{})

	if err := pipeline.insertWithRetry(context.Background(), config.WriterConfig{}, nil); err == nil {
		t.Fatal("zero attempts must not report a successful write")
	}
	if called {
		t.Fatal("processor was called with zero attempts")
	}
}

func TestInsertWithRetryStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	processor := &fakeProcessor{process: func(context.Context, []model.Log) error {
		calls++
		cancel()
		return context.DeadlineExceeded
	}}
	pipeline := NewPipeline(nil, processor, config.BatchConfig{}, retryConfig(3))

	if err := pipeline.insertWithRetry(ctx, retryConfig(3), nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected no retry after cancellation, got %d attempts", calls)
	}
}

func retryConfig(maxAttempts int) config.WriterConfig {
	return config.WriterConfig{
		InsertTimeout:     time.Second,
		MaxAttempts:       maxAttempts,
		RetryInitialDelay: time.Millisecond,
		RetryMaxDelay:     time.Millisecond,
	}
}
