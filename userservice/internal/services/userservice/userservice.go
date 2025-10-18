package userservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"userservice/internal/domain/models"
	"userservice/internal/storage"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidUserName    = errors.New("invalid user name")
	ErrInvalidUserSurname = errors.New("invalid user surname")
	ErrInvalidUserAvatar  = errors.New("invalid user avatar")
)

type UserService struct {
	log          *slog.Logger
	userProvider UserProvider
	userUpdater  UserUpdater
}

type UserProvider interface {
	GetUserByID(ctx context.Context, userID int64) (*models.User, error)
}

type UserUpdater interface {
	UpdateUser(ctx context.Context, user *models.User) (*models.User, error)
}

func New(log *slog.Logger, userProvider UserProvider, userUpdater UserUpdater) *UserService {
	return &UserService{
		log:          log,
		userProvider: userProvider,
		userUpdater:  userUpdater,
	}
}

func (us *UserService) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	const op = "userservice.GetUser"

	log := us.log.With(slog.String("op", op), slog.Int64("user_id", userID))
	log.Debug("getting user by id")

	user, err := us.userProvider.GetUserByID(ctx, userID)

	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: error user not found: %w", op, ErrUserNotFound)
		}
		log.Error("error get user", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: error get user: %w", op, err)
	}

	return user, nil
}

func (us *UserService) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	const op = "userservice.UpdateUser"
	log := us.log.With(slog.String("op", op), slog.Int64("user_id", user.ID))
	log.Debug("updating user")

	if err := validateUser(user); err != nil {
		if errors.Is(err, ErrInvalidUserAvatar) {
			log.Warn("invalid user avatar", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: invalid user avatar: %w", op, ErrInvalidUserAvatar)
		}
		if errors.Is(err, ErrInvalidUserName) {
			log.Warn("invalid user name", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: invalid user name: %w", op, ErrInvalidUserName)
		}
		if errors.Is(err, ErrInvalidUserSurname) {
			log.Warn("invalid user surname", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: invalid user surname: %w", op, ErrInvalidUserSurname)
		}

		log.Warn("invalid user data", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: invalid user data: %w", op, err)
	}
	updatedUser, err := us.userUpdater.UpdateUser(ctx, user)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: error user not found: %w", op, ErrUserNotFound)
		}
		log.Error("error update user", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: error update user: %w", op, err)
	}

	return updatedUser, nil
}

func validateUser(user *models.User) error {
	if user.Name == "" {
		return ErrInvalidUserName
	}
	if user.Surname == "" {
		return ErrInvalidUserSurname
	}
	if user.Avatar != nil && !IsSquareOrEmpty(user.Avatar) {
		return ErrInvalidUserAvatar
	}

	return nil
}

func IsSquareOrEmpty(avatar []byte) bool {
	if len(avatar) == 0 {
		return true
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(avatar))
	if err != nil {
		return false
	}

	return cfg.Width == cfg.Height
}
