package eventgetter

import (
	"context"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type EventConsumer interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

type EventProcessor interface {
	ProcessEvent(ctx context.Context, event []byte) error
}

type Getter struct {
	log            *slog.Logger
	EventConsumer  EventConsumer
	EventProcessor EventProcessor
}

func New(log *slog.Logger, consumer EventConsumer, processor EventProcessor) *Getter {
	return &Getter{
		log:            log,
		EventConsumer:  consumer,
		EventProcessor: processor,
	}
}

func (g *Getter) GetEventStart(ctx context.Context, handlePeriod time.Duration) error {
	const op = "eventgetter.Getter.GetEvent"

	log := g.log.With(slog.String("op", op))
	ticker := time.NewTicker(handlePeriod)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("stopping event getter")
				ticker.Stop()
				return
			case <-ticker.C:
				message, err := g.EventConsumer.ReadMessage(ctx)
				if err != nil {
					log.Error("failed to read message from consumer", slog.String("error", err.Error()))
					continue
				}

				log.Info("event received", slog.Int("message_size", len(message.Value)))

				err = g.EventProcessor.ProcessEvent(ctx, message.Value)
				if err != nil {
					log.Error("failed to process event", slog.String("error", err.Error()))
					continue
				}
				log.Info("event processed successfully")

				err = g.EventConsumer.CommitMessages(ctx, message)
				if err != nil {
					log.Error("failed to commit message", slog.String("error", err.Error()))
					continue
				}
				log.Info("message committed successfully")

				continue
			}
		}
	}()

	return nil
}
