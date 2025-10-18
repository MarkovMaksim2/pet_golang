package storage

import (
	"context"
	"errors"
	"userservice/internal/domain/models"
)

var (
	ErrUserNotFound      = errors.New("user not found in storage by id")
	ErrUserAlreadyExists = errors.New("user already exists in storage with given id")
)

type Storage interface {
	GetUserByID(ctx context.Context, userID int64) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
}
