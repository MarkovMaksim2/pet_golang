package app

import (
	"log/slog"
	"os"
	grpcapp "userservice/internal/app/grpc"
	"userservice/internal/services/userservice"
	"userservice/internal/storage/sqlstorage"
)

type App struct {
	GRPCApp *grpcapp.App
}

func New(log *slog.Logger, grpcPort int, storagePath string, secret string) *App {
	storage, err := sqlstorage.New("sqlite3", storagePath)
	if err != nil {
		log.Error("failed to init storage", slog.String("error", err.Error()))
		os.Exit(1)
	}

	userService := userservice.New(log, storage, storage)
	gRPCApp := grpcapp.New(log, userService, grpcPort, secret)

	return &App{
		GRPCApp: gRPCApp,
	}
}
