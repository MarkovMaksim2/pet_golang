package sqlstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"userservice/internal/domain/models"
	"userservice/internal/storage"

	_ "github.com/mattn/go-sqlite3"
)

type Dialect int

const (
	Postgres Dialect = iota
	MySQL
	SQLite
)

type SQLStorage struct {
	db      *sql.DB
	dialect Dialect
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

	dialect := detectDialect(driver)
	log.Info("database connected", slog.String("dialect", driver))
	return &SQLStorage{
		db:      db,
		dialect: dialect,
	}, nil
}

func detectDialect(driver string) Dialect {
	switch driver {
	case "postgres", "pgx":
		return Postgres
	case "mysql":
		return MySQL
	default:
		return SQLite
	}
}

func (s *SQLStorage) placeholder(n int) string {
	switch s.dialect {
	case Postgres:
		return fmt.Sprintf("$%d", n)
	default:
		return "?"
	}
}

func (s *SQLStorage) GetUserByID(ctx context.Context, userId int64) (*models.User, error) {
	const op = "sqlstorage.GetUserById"

	q := fmt.Sprintf(`SELECT id, name, surname, avatar FROM users WHERE id = %s`, s.placeholder(1))
	stmt, err := s.db.Prepare(q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, userId)

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
	q := fmt.Sprintf(`UPDATE users SET name = %s, surname = %s, avatar = %s WHERE id = %s RETURNING id, name, surname, avatar`,
		s.placeholder(1), s.placeholder(2), s.placeholder(3), s.placeholder(4))
	stmt, err := s.db.Prepare(q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, user.Name, user.Surname, user.Avatar, user.ID)

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
	q := fmt.Sprintf(`
	INSERT INTO users (id, name, surname, avatar)
	VALUES (%s, %s, %s, %s)
	ON CONFLICT (id) DO NOTHING
	RETURNING id, name, surname, avatar`,
		s.placeholder(1), s.placeholder(2), s.placeholder(3), s.placeholder(4))
	stmt, err := s.db.Prepare(q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, user.ID, user.Name, user.Surname, user.Avatar)
	var u models.User
	if err := row.Scan(&u.ID, &u.Name, &u.Surname, &u.Avatar); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrUserAlreadyExists)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &u, nil
}
