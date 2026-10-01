package producer

import (
	"LogFlux/contracts/events"
	"LogFlux/services/ingestion/internal/config"
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(config *config.KafkaConfig) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(config.Brokers...),
			Topic:        config.Topic,
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
			BatchTimeout: config.BatchTimeout,
			WriteTimeout: config.WriteTimeout,
		},
	}
}

func (p *KafkaProducer) PublishLogs(ctx context.Context, events []events.LogAccepted) error {
	if len(events) == 0 {
		return nil
	}

	messages := make([]kafka.Message, 0, len(events))
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("error encoding event %s: %w", event.EventID, err)
		}

		messages = append(messages, kafka.Message{
			Key:   []byte(event.ProjectID + ":" + event.Service),
			Value: data,
		})
	}

	err := p.writer.WriteMessages(ctx, messages...)
	if err != nil {
		return fmt.Errorf("error publish logs: %w", err)
	}
	return nil
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
