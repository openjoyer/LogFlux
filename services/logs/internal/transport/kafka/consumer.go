package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewConsumer(
	brokers []string,
	topic string,
	groupID string,
	clientId string,
	maxBytes int,
) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			CommitInterval: 0,
			MinBytes:       maxBytes,
			MaxBytes:       maxBytes,
			Dialer: &kafka.Dialer{
				ClientID: clientId,
				Timeout:  10 * time.Second,
			},
		}),
	}
}

func (c *KafkaConsumer) Fetch(ctx context.Context) (kafka.Message, error) {
	return c.reader.FetchMessage(ctx)
}

func (c *KafkaConsumer) Commit(ctx context.Context, messages ...kafka.Message) error {
	return c.reader.CommitMessages(ctx, messages...)
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
