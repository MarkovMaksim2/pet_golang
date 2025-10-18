package storage

import "errors"

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUserNotExists = errors.New("user already exists")
	ErrAppNotFound   = errors.New("app not found")
	ErrUserExists    = errors.New("user already exists")
	ErrNoNewEvents   = errors.New("no new events")
)
