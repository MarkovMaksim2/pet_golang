package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sso/internal/domain/models"
	"sso/internal/storage"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

type eventPayload struct {
	Id    int64  `json:"id"`
	Email string `json:"email"`
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveUser(ctx context.Context, email string, passHash []byte) (resID int64, err error) {
	const op = "storage.sqlite.SaveUser"

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		commitErr := tx.Commit()
		if commitErr != nil {
			err = fmt.Errorf("%s: commit tx: %w", op, commitErr)
		}
	}()

	query, args, err := sq.Insert("users").Columns("email", "pass_hash").Values(email, passHash).ToSql()
	if err != nil {
		return 0, fmt.Errorf("%s: build query: %w", op, err)
	}

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		var sqliteErr sqlite3.Error

		if errors.As(err, &sqliteErr) &&
			sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	resID, err = res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	eventPayloadBytes, err := json.Marshal(eventPayload{
		Id:    resID,
		Email: email,
	})
	if err != nil {
		return resID, fmt.Errorf("failed to marshal event: %w", err)
	}
	eventPayload := string(eventPayloadBytes)

	if err := s.SaveEvent(ctx, tx, "UserCreated", eventPayload); err != nil {
		return 0, fmt.Errorf("%s: save event: %w", op, err)
	}

	return resID, nil
}

func (s *Storage) SaveEvent(ctx context.Context, tx *sql.Tx, eventType, payload string) error {
	const op = "storage.sqlite.SaveEvent"

	query, args, err := sq.Insert("messages").Columns("event_type", "payload").Values(eventType, payload).ToSql()
	if err != nil {
		return fmt.Errorf("%s: build query: %w", op, err)
	}

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.sqlite.User"
	var user models.User

	query, args, err := sq.Select("id", "email", "pass_hash").From("users").Where(sq.Eq{"email": email}).ToSql()
	if err != nil {
		return user, fmt.Errorf("%s: build query: %w", op, err)
	}

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return user, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, args...)
	err = row.Scan(&user.ID, &user.Email, &user.PassHash)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotExists)
		}

		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "storage.sqlite.IsAdmin"

	query, args, err := sq.Select("is_admin").From("admins").Where(sq.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return false, fmt.Errorf("%s: build query: %w", op, err)
	}

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, args...)
	var isAdmin bool
	err = row.Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return false, fmt.Errorf("%s: %w", op, err)
	}

	return isAdmin, nil
}

func (s *Storage) App(ctx context.Context, appID int64) (models.App, error) {
	const op = "storage.sqlite.App"
	var app models.App

	query, args, err := sq.Select("id", "name", "secret").From("apps").Where(sq.Eq{"id": appID}).ToSql()
	if err != nil {
		return models.App{}, fmt.Errorf("%s: build query: %w", op, err)
	}

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, args...)
	err = row.Scan(&app.ID, &app.Name, &app.Secret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (s *Storage) GetNewEvent(ctx context.Context) (models.Event, error) {
	const op = "storage.sqlite.GetNewEvent"

	query, args, err := sq.Select("id", "event_type", "payload").
		From("messages").
		Where(sq.Eq{"status": "new"}).
		OrderBy("created_at").
		Limit(1).ToSql()
	if err != nil {
		return models.Event{}, fmt.Errorf("%s: build query: %w", op, err)
	}

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return models.Event{}, fmt.Errorf("%s: %w", op, err)
	}

	row := stmt.QueryRowContext(ctx, args...)

	var event models.Event
	err = row.Scan(&event.ID, &event.Type, &event.Payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Event{}, fmt.Errorf("%s: %w", op, storage.ErrNoNewEvents)
		}
		return models.Event{}, fmt.Errorf("%s: %w", op, err)
	}

	return event, nil
}

func (s *Storage) MarkEventAsDone(ctx context.Context, eventID int64) error {
	const op = "storage.sqlite.MarkEventAsDone"

	query, args, err := sq.Update("messages").Set("status", "sent").Where(sq.Eq{"id": eventID}).ToSql()
	if err != nil {
		return fmt.Errorf("%s: build query: %w", op, err)
	}

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
