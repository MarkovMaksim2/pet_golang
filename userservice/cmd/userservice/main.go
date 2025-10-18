package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	"userservice/internal/app"
	"userservice/internal/config"
	kafkaconsumer "userservice/internal/lib/kafka"
	"userservice/internal/lib/metrics"
	eventgetter "userservice/internal/services/event-getter"
	"userservice/internal/services/processors"
	"userservice/internal/storage/sqlstorage"
)

const (
	envLocal       = "local"
	envDevelopment = "development"
	envProduction  = "production"
)

func main() {
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)

	userProcessor := setupUserProcessor(log, cfg.StoragePath)
	kafkaConsumer := setupKafkaConsumer(log, cfg)
	defer kafkaConsumer.Close()

	userEventGetter := eventgetter.New(log, kafkaConsumer, userProcessor)
	userEventGetter.GetEventStart(context.Background(), 5*time.Second)
	go func() {
		if err := metrics.Listen(cfg.Metrics.Host, cfg.Metrics.Port); err != nil {
			log.Error("failed to start metrics server", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	log.Info("starting application")

	application := app.New(
		log,
		cfg.GRPC.Port,
		cfg.StoragePath,
		cfg.Secret,
	)

	go application.GRPCApp.MustRun()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	stopSignal := <-stop
	log.Info("shutting down application", slog.String("signal", stopSignal.String()))

	application.GRPCApp.Stop()
	// kafkaconsumer.Close()

	log.Info("application stopped")
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

func setupKafkaConsumer(log *slog.Logger, cfg *config.Config) *kafkaconsumer.Consumer {
	kafkaconsumer, error := kafkaconsumer.New(
		log, cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, cfg.Kafka.DialAddr)
	if error != nil {
		log.Error("failed to initialize kafka consumer")
		os.Exit(1)
	}

	return kafkaconsumer
}

func setupUserProcessor(log *slog.Logger, storagePath string) *processors.UserProcessor {
	storage, err := sqlstorage.New("sqlite3", storagePath)
	if err != nil {
		log.Error("failed to initialize storage")
		os.Exit(1)
	}
	return processors.NewUserProcessor(log, storage)
}
