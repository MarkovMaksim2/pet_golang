package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sso/internal/app"
	"sso/internal/config"
	kafkaproducer "sso/internal/lib/kafka"
	eventsender "sso/internal/services/event-sender"
	"sso/internal/storage/sqlite"
	"syscall"
	"time"
)

const (
	envLocal       = "local"
	envDevelopment = "development"
	envProduction  = "production"
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)

	log.Info("starting application")

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to create storage", slog.String("error", err.Error()))
		os.Exit(1)
	}
	kafkaproducer, err := kafkaproducer.New(
		log, cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.DialAdress)
	if err != nil {
		log.Error("failed to create storage", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer kafkaproducer.Close()

	eventSender := eventsender.New(log, storage, kafkaproducer)
	go eventSender.StartProcessingEvents(context.Background(), 5*time.Second)

	application := app.New(
		log,
		cfg.GRPC.Port,
		cfg.StoragePath,
		cfg.TokenTTL,
	)

	go application.GRPCApp.MustRun()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	stopSignal := <-stop
	log.Info("shutting down application", slog.String("signal", stopSignal.String()))

	application.GRPCApp.Stop()

	log.Info("application stopped")
	// TODO: Implement SSO command functionality
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelDebug},
			),
		)
	case envDevelopment:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelDebug},
			),
		)
	case envProduction:
		log = slog.New(
			slog.NewJSONHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelInfo},
			),
		)
	}

	return log
}
