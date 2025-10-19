package kafkaconsumer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

var (
	ErrNoBrokers = fmt.Errorf("No kafka brokers provided")
	ErrNoTopic   = fmt.Errorf("No kafka topic provided")
)

type Consumer struct {
	log    *slog.Logger
	reader *kafka.Reader
}

func New(
	log *slog.Logger,
	brokers []string,
	topic string,
	groupID string,
	dialAddr string) (*Consumer, error) {
	if len(brokers) == 0 {
		return nil, ErrNoBrokers
	}
	if topic == "" {
		return nil, ErrNoTopic
	}

	conn, err := kafka.Dial("tcp", dialAddr)
	if err != nil {
		log.Error("failed to dial Kafka")
		return nil, fmt.Errorf("dial kafka: %w", err)
	}
	defer conn.Close()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})

	log.Info("Kafka consumer initialized", slog.String("topic", topic))

	return &Consumer{
		log:    log,
		reader: reader,
	}, nil
}

func (c *Consumer) Close() error {
	const op = "kafkaconsumer.Close"

	c.log.With(slog.String("op", op)).
		Info("closing Kafka producer")
	return c.reader.Close()
}

func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	const op = "kafkaconsumer.ReadMessage"

	log := c.log.With(slog.String("op", op))

	msg, err := c.reader.ReadMessage(ctx)

	if err != nil {
		log.Error("failed to read message from Kafka", slog.String("error", err.Error()))
		return kafka.Message{}, fmt.Errorf("read message: %w", err)
	}

	log.Info("message read from Kafka", slog.Int64("offset", msg.Offset))

	return msg, nil
}

func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	const op = "kafkaconsumer.CommitMessages"

	log := c.log.With(slog.String("op", op))

	err := c.reader.CommitMessages(ctx, msgs...)
	if err != nil {
		log.Error("failed to commit message", slog.String("error", err.Error()))
		return fmt.Errorf("commit message: %w", err)
	}

	return nil
}
