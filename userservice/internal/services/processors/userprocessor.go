package processors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"userservice/internal/domain/models"
	"userservice/internal/storage"
)

type UserPayload struct {
	UserID int64  `json:"id"`
	Email  string `json:"email"`
}

type UserProcessor struct {
	log     *slog.Logger
	storage storage.Storage
	Processor
}

func NewUserProcessor(log *slog.Logger, storage storage.Storage) *UserProcessor {
	return &UserProcessor{
		log:     log,
		storage: storage,
	}
}

func (p *UserProcessor) ProcessEvent(ctx context.Context, payload []byte) error {
	const op = "userprocessor.UserProcessor.ProcessEvent"

	log := p.log.With(slog.String("op", op))

	userPayload, err := parseUserPayload(payload)
	if err != nil {
		log.Error("failed to parse user payload", slog.String("error", err.Error()))
		return fmt.Errorf("parse error %w", err)
	}

	user := &models.User{
		ID:      userPayload.UserID,
		Name:    "no name",
		Surname: "no surname",
		Avatar:  []byte{},
	}
	_, err = p.storage.CreateUser(ctx, user)
	if err != nil {
		if errors.Is(err, storage.ErrUserAlreadyExists) {
			log.Info("user already exists, skipping creation", slog.Int64("user_id", userPayload.UserID))
			return nil
		}
		log.Error("failed to create user", slog.Int64("user_id", userPayload.UserID), slog.String("error", err.Error()))
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func parseUserPayload(payload []byte) (*UserPayload, error) {
	var event *UserPayload = &UserPayload{}

	err := json.Unmarshal([]byte(payload), event)

	if err != nil {
		return nil, err
	}

	return event, nil
}
