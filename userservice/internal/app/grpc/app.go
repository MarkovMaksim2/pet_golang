package grpcapp

import (
	"fmt"
	"log/slog"
	"net"

	userservicegrpc "userservice/internal/grpc/usersevice"
	"userservice/internal/middleware"

	"google.golang.org/grpc"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

func New(logger *slog.Logger, userService userservicegrpc.UserService, port int, secret string) *App {
	jwtInterceptor := grpc.UnaryInterceptor(middleware.JWTAuthInterceptor(secret))
	gRPCServer := grpc.NewServer(jwtInterceptor)

	userservicegrpc.Register(gRPCServer, userService)

	return &App{
		log:        logger,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		a.log.Error("failed to run gRPC server", slog.String("error", err.Error()))
		panic(err)
	}
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
