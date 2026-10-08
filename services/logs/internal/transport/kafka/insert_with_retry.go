package kafka

import (
	"LogFlux/services/logs/internal/config"
	"LogFlux/services/logs/internal/model"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"time"
)

func (p *Pipeline) insertWithRetry(ctx context.Context, cfg config.WriterConfig, logs []model.Log) error {
	if cfg.MaxAttempts < 1 {
		return fmt.Errorf("max attempts must be positive")
	}

	delay := cfg.RetryInitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		attemptCtx, cancel := context.WithTimeout(ctx, cfg.InsertTimeout)
		err := p.processor.Process(attemptCtx, logs)
		cancel()

		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var netErr net.Error
		retryable := errors.Is(err, context.DeadlineExceeded) ||
			errors.As(err, &netErr)

		if !retryable || attempt == cfg.MaxAttempts {
			return fmt.Errorf(
				"write failed after %d attempt(s): %w",
				attempt, err,
			)
		}

		timeout := delay/2 + time.Duration(rand.Int64N(int64(delay/2)+1))

		timer := time.NewTimer(timeout)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		delay = min(delay*2, cfg.RetryMaxDelay)
	}

	return nil
}
