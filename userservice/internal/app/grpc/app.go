package grpcapp

import (
	"fmt"
	"log/slog"
	"net"

	userservicegrpc "userservice/internal/grpc/usersevice"
	"userservice/internal/middleware"
	"userservice/internal/services/userservice"
	"userservice/internal/storage/sqlstorage"

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
	Secret      string
}

func New(log *slog.Logger, appConfig AppConfig) (*App, error) {
	storage, err := sqlstorage.New("sqlite3", appConfig.StoragePath)
	if err != nil {
		log.Error("failed to init storage", slog.String("error", err.Error()))
		return &App{}, fmt.Errorf("storage creation: %w", err)
	}

	userService := userservice.New(log, storage, storage)
	jwtInterceptor := grpc.UnaryInterceptor(middleware.JWTAuthInterceptor(appConfig.Secret))
	gRPCServer := grpc.NewServer(jwtInterceptor)

	userservicegrpc.Register(gRPCServer, userService)

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
		log.Error("failed to listen", slog.String("error", err.Error()))
		return fmt.Errorf("%s: failed to listen: %w", op, err)
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
