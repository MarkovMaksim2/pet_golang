package sqlstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"userservice/internal/domain/models"
	"userservice/internal/storage"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
)

type SQLStorage struct {
	db *sql.DB
	storage.Storage
}

func New(driver string, dsn string) (*SQLStorage, error) {
	const op = "sqlstorage.New"
	log := slog.With(slog.String("op", op), slog.String("driver", driver))
	db, err := sql.Open(driver, dsn)

	if err != nil {
		return nil, fmt.Errorf("%s: failed to open database: %w", op, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: failed to ping database: %w", op, err)
	}

	log.Info("database connected", slog.String("dialect", driver))
	return &SQLStorage{
		db: db,
	}, nil
}

func (s *SQLStorage) GetUserByID(ctx context.Context, userId int64) (*models.User, error) {
	const op = "sqlstorage.GetUserById"

	query, args, err := sq.Select("id", "name", "surname", "avatar").From("users").Where(sq.Eq{"id": userId}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build query: %w", op, err)
	}

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, args...)

	var u models.User
	if err := row.Scan(&u.ID, &u.Name, &u.Surname, &u.Avatar); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: user not found: %w", op, storage.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &u, nil
}

func (s *SQLStorage) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	const op = "sqlstorage.UpdateUser"
	query, args, err := sq.Update("users").SetMap(sq.Eq{
		"name":    user.Name,
		"surname": user.Surname,
		"avatar":  user.Avatar,
	}).Where(sq.Eq{"id": user.ID}).Suffix("RETURNING id, name, surname, avatar").ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build query: %w", op, err)
	}
	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, args...)

	var u models.User
	if err := row.Scan(&u.ID, &u.Name, &u.Surname, &u.Avatar); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: user not found: %w", op, storage.ErrUserNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &u, nil
}

func (s *SQLStorage) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	const op = "sqlstorage.CreateUser"

	query, args, err := sq.Insert("users").Columns("id", "name", "surname", "avatar").
		Values(user.ID, user.Name, user.Surname, user.Avatar).
		Suffix("ON CONFLICT (id) DO NOTHING RETURNING id, name, surname, avatar").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: build query: %w", op, err)
	}
	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, args...)
	var u models.User
	if err := row.Scan(&u.ID, &u.Name, &u.Surname, &u.Avatar); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrUserAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &u, nil
}
