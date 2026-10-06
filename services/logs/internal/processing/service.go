package processing

import (
	"LogFlux/services/logs/internal/model"
	"context"
)

type LogWriter interface {
	InsertBatch(ctx context.Context, logs []model.Log) error
}

type LogProcessor struct {
	writer LogWriter
}

func NewLogProcessor(writer LogWriter) *LogProcessor {
	return &LogProcessor{
		writer: writer,
	}
}

func (p *LogProcessor) Process(ctx context.Context, logs []model.Log) error {
	return p.writer.InsertBatch(ctx, logs)
}
