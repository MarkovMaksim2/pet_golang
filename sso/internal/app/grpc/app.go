package grpcapp

import (
	"fmt"
	"log/slog"
	"net"
	authgrpc "sso/internal/grpc/auth"
	"sso/internal/services/auth"
	"sso/internal/storage/sqlite"
	"time"

	"google.golang.org/grpc"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

type AppConfig struct {
	GrpcPort    int
	StoragePath string
	TokenTTL    time.Duration
}

func New(log *slog.Logger, appConfig AppConfig) (*App, error) {
	storage, err := sqlite.New(appConfig.StoragePath)
	if err != nil {
		return &App{}, fmt.Errorf("create storage: %w", err)
	}

	authService := auth.New(log, storage, storage, storage, appConfig.TokenTTL)
	gRPCServer := grpc.NewServer()

	authgrpc.Register(gRPCServer, authService)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       appConfig.GrpcPort,
	}, nil
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(slog.String("op", op), slog.Int("port", a.port))

	log.Info("starting server")

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: listen: %w", op, err)
	}

	log.Info("server started", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: serve: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).
		Info("stopping server", slog.Int("port", a.port))
	a.gRPCServer.GracefulStop()
}
