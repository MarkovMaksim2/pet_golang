package kafkaproducer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

type Producer struct {
	log    *slog.Logger
	writer *kafka.Writer
}

func New(
	log *slog.Logger,
	brokers []string,
	topic string,
	dialAdress string) (*Producer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("no Kafka brokers provided")
	}
	if topic == "" {
		return nil, fmt.Errorf("no Kafka topic provided")
	}

	conn, err := kafka.Dial("tcp", dialAdress)
	if err != nil {
		log.Error("failed to dial Kafka")
		return nil, fmt.Errorf("dial Kafka: %w", err)
	}
	defer conn.Close()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		log.Error("failed to create topic:", slog.String("error", err.Error()))
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      brokers,
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	})

	log.Info("Kafka producer initialized", slog.String("topic", topic))

	return &Producer{
		log:    log,
		writer: writer,
	}, nil
}

func (p *Producer) Send(ctx context.Context, key, value []byte) error {
	const op = "kafkaproducer.Send"

	log := p.log.With(slog.String("op", op))
	err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	})
	if err != nil {
		log.Error("failed to send Kafka message", slog.Any("error", err))
		return fmt.Errorf("send message: %w", err)
	}

	log.Debug("Kafka message sent",
		slog.String("key", string(key)),
		slog.Int("value_size", len(value)),
	)
	return nil
}

func (p *Producer) Close() error {
	const op = "kafkaproducer.Close"

	p.log.With(slog.String("op", op)).
		Info("closing Kafka producer")
	return p.writer.Close()
}
