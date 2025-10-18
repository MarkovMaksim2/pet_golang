package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sso/internal/domain/models"
	"sso/internal/lib/jwt"
	"sso/internal/storage"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	emptyUser = 0
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrAppNotFound        = errors.New("app not found")
	ErrUserNotFound       = errors.New("user not found")
)

type Auth struct {
	log          *slog.Logger
	userSaver    UserSaver
	userProvider UserProvider
	appProvider  AppProvider
	tokenTTL     time.Duration
}

type UserSaver interface {
	SaveUser(
		ctx context.Context,
		email string,
		passHash []byte,
	) (uid int64, err error)
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int64) (models.App, error)
}

// New returns a new Auth service.
func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	appProvider AppProvider,
	tokenTTL time.Duration,
) *Auth {
	return &Auth{
		log:          log,
		userSaver:    userSaver,
		userProvider: userProvider,
		appProvider:  appProvider,
		tokenTTL:     tokenTTL,
	}
}

// Login checks user credentials and returns a JWT token if successful.
func (a *Auth) Login(
	ctx context.Context,
	email string,
	password string,
	appID int,
) (string, error) {
	const op = "auth.Login"
	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("user logging in")

	user, err := a.userProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", slog.String("error", err.Error()))
			return "", fmt.Errorf("%s: get user: %w", op, ErrInvalidCredentials)
		}
		log.Error("failed to get user", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: get user: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		log.Warn("invalid password", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: compare password: %w", op, ErrInvalidCredentials)
	}

	app, err := a.appProvider.App(ctx, int64(appID))
	if err != nil {
		if errors.Is(err, storage.ErrAppNotFound) {
			log.Warn("app not found", slog.String("error", err.Error()))
			return "", fmt.Errorf("%s: get app: %w", op, ErrAppNotFound)
		}
		log.Error("failed to get app", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: get app: %w", op, err)
	}
	log.Info("login successful", slog.Int64("user_id", user.ID), slog.Int64("app_id", app.ID))

	token, err := jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		log.Error("failed to create token", slog.String("error", err.Error()))
		return "", fmt.Errorf("%s: create token: %w", op, err)
	}
	return token, nil
}

// RegisterNewUser creates a new user with the given email and password.
func (a *Auth) RegisterNewUser(
	ctx context.Context,
	email string,
	password string,
) (int64, error) {
	const op = "auth.RegisterNewUser"

	log := a.log.With(
		slog.String("op", op),
		slog.String("email", email),
	)

	log.Info("registering new user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		log.Error("failed to hash password", slog.String("error", err.Error()))

		return emptyUser, fmt.Errorf("%s: hash password: %w", op, err)
	}

	uid, err := a.userSaver.SaveUser(ctx, email, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Warn("user already exists", slog.String("error", err.Error()))
			return emptyUser, fmt.Errorf("%s: save user: %w", op, ErrUserExists)
		}
		log.Error("failed to save user", slog.String("error", err.Error()))

		return emptyUser, fmt.Errorf("%s: save user: %w", op, err)
	}

	log.Info("user registered successfully", slog.Int64("user_id", uid))
	return uid, nil
}

// IsAdmin checks if the user with the given ID has admin privileges.
func (a *Auth) IsAdmin(
	ctx context.Context,
	userID int64,
) (bool, error) {
	const op = "auth.IsAdmin"

	log := a.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
	)

	log.Info("checking admin privileges")

	isAdmin, err := a.userProvider.IsAdmin(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", slog.String("error", err.Error()))
			return false, fmt.Errorf("%s: check admin: %w", op, ErrUserNotFound)
		}
		log.Error("failed to check admin privileges", slog.String("error", err.Error()))
		return false, fmt.Errorf("%s: check admin: %w", op, err)
	}

	log.Info("admin check successful", slog.Bool("is_admin", isAdmin))

	return isAdmin, nil
}
