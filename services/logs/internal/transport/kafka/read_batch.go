package kafka

import (
	"LogFlux/contracts/events"
	"LogFlux/services/logs/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func (p *Pipeline) readBatch(ctx context.Context) (Batch, error) {
	var batch Batch
	var size int

	readCtx := ctx
	var cancel context.CancelFunc

	defer func() {
		if cancel != nil {
			cancel()
		}
	}()

	for {
		msg, err := p.consumer.Fetch(readCtx)
		if err != nil {
			if ctx.Err() != nil {
				return Batch{}, ctx.Err()
			}

			if len(batch.Messages) > 0 &&
				errors.Is(err, context.DeadlineExceeded) &&
				errors.Is(readCtx.Err(), context.DeadlineExceeded) {
				return batch, nil
			}

			return Batch{}, fmt.Errorf("fetch message: %w", err)
		}

		if len(batch.Messages) == 0 {
			readCtx, cancel = context.WithTimeout(ctx, p.batchCfg.FlushInterval)
		}

		var event events.LogAccepted
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return Batch{}, fmt.Errorf(
				"decode partition=%d offset=%d: %w",
				msg.Partition, msg.Offset, err,
			)
		}

		batch.Logs = append(batch.Logs, model.Log{
			EventID:    event.EventID,
			ProjectID:  event.ProjectID,
			Timestamp:  event.Timestamp,
			ReceivedAt: event.AcceptedAt,
			Service:    event.Service,
			Level:      event.Level,
			Message:    event.Message,
			Attributes: event.Attributes,
		})
		batch.Messages = append(batch.Messages, msg)

		size += len(msg.Key) + len(msg.Value)

		if ctx.Err() != nil {
			return Batch{}, ctx.Err()
		}

		if len(batch.Logs) >= p.batchCfg.MaxEvents || size >= p.batchCfg.MaxBytes || readCtx.Err() != nil {
			return batch, nil
		}
	}
}
