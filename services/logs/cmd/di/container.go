package di

import (
	"LogFlux/services/logs/internal/config"
	"LogFlux/services/logs/internal/processing"
	"LogFlux/services/logs/internal/repository/clickhouse"
	"LogFlux/services/logs/internal/transport/kafka"
	"context"
	"errors"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Container struct {
	cfg config.Config

	conn     driver.Conn
	consumer kafka.Consumer

	writer    *clickhouse.Writer
	processor *processing.LogProcessor
	pipeline  *kafka.Pipeline
}

func NewContainer(ctx context.Context, cfg config.Config) (*Container, error) {
	conn, err := clickhouse.NewClient(ctx, cfg.ClickHouse)
	if err != nil {
		return nil, err
	}

	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.Topic,
		cfg.Kafka.GroupID,
		cfg.Kafka.ClientID,
		cfg.Kafka.MaxBytes,
	)

	return &Container{
		cfg:      cfg,
		conn:     conn,
		consumer: consumer,
	}, nil
}

func (c *Container) Writer() *clickhouse.Writer {
	if c.writer == nil {
		c.writer = clickhouse.NewWriter(c.conn)
	}
	return c.writer
}

func (c *Container) Processor() *processing.LogProcessor {
	if c.processor == nil {
		c.processor = processing.NewLogProcessor(c.Writer())
	}
	return c.processor
}

func (c *Container) Pipeline() *kafka.Pipeline {
	if c.pipeline == nil {
		c.pipeline = kafka.NewPipeline(
			c.consumer,
			c.Processor(),
			c.cfg.Batch,
			c.cfg.Writer,
		)
	}
	return c.pipeline
}

func (c *Container) Close() error {
	consumerErr := c.consumer.Close()
	clickHouseErr := c.conn.Close()

	return errors.Join(consumerErr, clickHouseErr)
}
