package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sso/internal/domain/models"
	"sso/internal/storage"

	"github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

// New creates a new SQLite storage instance.
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

	stmt, err := tx.Prepare("INSERT INTO users (email, pass_hash) VALUES (?, ?)")
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, email, passHash)
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

	eventPayload := fmt.Sprintf(`{"id": %d, "email": "%s"}`, resID, email)

	if err := s.SaveEvent(tx, "UserCreated", eventPayload); err != nil {
		return 0, fmt.Errorf("%s: save event: %w", op, err)
	}

	return resID, nil
}

func (s *Storage) SaveEvent(tx *sql.Tx, eventType, payload string) error {
	const op = "storage.sqlite.SaveEvent"

	stmt, err := tx.Prepare("INSERT INTO messages (event_type, payload) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	defer stmt.Close()
	_, err = stmt.Exec(eventType, payload)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.sqlite.User"
	var user models.User

	stmt, err := s.db.Prepare("SELECT id, email, pass_hash FROM users WHERE email = ?")
	if err != nil {
		return user, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, email)
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

	stmt, err := s.db.Prepare("SELECT is_admin FROM admins WHERE user_id = ?")
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, userID)
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

	stmt, err := s.db.Prepare("SELECT id, name, secret FROM apps WHERE id = ?")
	if err != nil {
		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, appID)
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

	row := s.db.QueryRow("SELECT id, event_type, payload FROM messages WHERE status = 'new' ORDER BY created_at LIMIT 1")

	var event models.Event
	err := row.Scan(&event.ID, &event.Type, &event.Payload)
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

	stmt, err := s.db.Prepare("UPDATE messages SET status = 'sent' WHERE id = ?")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	defer stmt.Close()
	_, err = stmt.Exec(eventID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
