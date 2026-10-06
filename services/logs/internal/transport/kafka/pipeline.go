package kafka

import (
	"LogFlux/services/logs/internal/config"
	"LogFlux/services/logs/internal/model"
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type Batch struct {
	Logs     []model.Log
	Messages []kafka.Message
}

type Processor interface {
	Process(ctx context.Context, logs []model.Log) error
}

type Consumer interface {
	Fetch(ctx context.Context) (kafka.Message, error)
	Commit(ctx context.Context, messages ...kafka.Message) error
	Close() error
}

type Pipeline struct {
	consumer  Consumer
	processor Processor
	batchCfg  config.BatchConfig
	writerCfg config.WriterConfig
}

func NewPipeline(
	consumer Consumer,
	processor Processor,
	batchConfig config.BatchConfig,
	writerConfig config.WriterConfig,
) *Pipeline {
	return &Pipeline{
		consumer:  consumer,
		processor: processor,
		batchCfg:  batchConfig,
		writerCfg: writerConfig,
	}
}

func (p *Pipeline) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	batches := make(chan Batch, 2)
	results := make(chan error, 2)

	go func() {
		defer close(batches)
		err := p.readWorker(ctx, batches)
		if err != nil {
			cancel()
		}
		results <- err
	}()

	go func() {
		defer close(batches)
		err := p.writeWorker(ctx, batches)
		if err != nil {
			cancel()
		}
		results <- err
	}()

	var firstErr error
	for range 2 {
		if err := <-results; err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (p *Pipeline) readWorker(ctx context.Context, batches chan<- Batch) error {
	for {
		batch, err := p.readBatch(ctx)
		if err != nil {
			return err
		}

		if len(batch.Messages) == 0 {
			continue
		}

		select {
		case batches <- batch:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *Pipeline) writeWorker(ctx context.Context, batches <-chan Batch) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batch, ok := <-batches:
			if !ok {
				return nil
			}

			if err := p.insertWithRetry(ctx, p.writerCfg, batch.Logs); err != nil {
				return fmt.Errorf("insert batch: %w", err)
			}

			if err := p.consumer.Commit(ctx, batch.Messages...); err != nil {
				return fmt.Errorf("commit batch: %w", err)
			}
		}
	}
}
