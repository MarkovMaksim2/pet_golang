package eventsender

import (
	"context"
	"errors"
	"log/slog"
	"sso/internal/domain/models"
	"sso/internal/storage"
	"time"
)

type EventProcessor interface {
	GetNewEvent(ctx context.Context) (models.Event, error)
	MarkEventAsDone(ctx context.Context, eventID int64) error
}

type EventProducer interface {
	Send(ctx context.Context, key, value []byte) error
}

type Sender struct {
	EventProcessor EventProcessor
	EventProducer  EventProducer
	log            *slog.Logger
}

func New(log *slog.Logger, eventProcessor EventProcessor, eventProducer EventProducer) *Sender {
	return &Sender{
		EventProcessor: eventProcessor,
		EventProducer:  eventProducer,
		log:            log,
	}
}

func (s *Sender) StartProcessingEvents(ctx context.Context, handlePeriod time.Duration) error {
	const op = "eventsender.StartProcessingEvents"

	log := s.log.With(slog.String("op", op))

	ticker := time.NewTicker(handlePeriod)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("stopping event processing")
				ticker.Stop()
				return
			case <-ticker.C:
			}
			event, err := s.EventProcessor.GetNewEvent(ctx)
			if err != nil {
				if errors.Is(err, storage.ErrNoNewEvents) {
					continue
				}
				log.Error("failed to get new event", slog.String("error", err.Error()))
				continue
			}

			if event.ID == 0 {
				log.Debug("no new events")
				continue
			}

			sendErr := s.EventProducer.Send(ctx, []byte(event.Type), []byte(event.Payload))
			if sendErr != nil {
				log.Error("failed to send event to producer",
					slog.Int64("event_id", event.ID),
					slog.String("error", sendErr.Error()))
				continue
			}

			if err := s.EventProcessor.MarkEventAsDone(ctx, event.ID); err != nil {
				log.Error(
					"failed to mark event as sent",
					slog.Int64("event_id", event.ID),
					slog.String("error", err.Error()),
				)
				continue
			}
		}
	}()

	return nil
}
