package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	grpcapp "userservice/internal/app/grpc"
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
	exitCode := 0

	defer func() {
		os.Exit(exitCode)
	}()

	application, err := grpcapp.New(
		log,
		grpcapp.AppConfig{
			GrpcPort:    cfg.GRPC.Port,
			StoragePath: cfg.StoragePath,
			Secret:      cfg.Secret,
		},
	)
	if err != nil {
		log.Error("failed to init app", slog.String("error", err.Error()))
		exitCode = 1
		return
	}
	defer application.Stop()

	userProcessor := setupUserProcessor(log, cfg.StoragePath)
	kafkaConsumer := setupKafkaConsumer(log, cfg)
	defer func() {
		if err := kafkaConsumer.Close(); err != nil {
			log.Error("Kafka close error")
			exitCode = 1
		}
	}()

	userEventGetter := eventgetter.New(log, kafkaConsumer, userProcessor)

	var wg sync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	go func() {
		defer wg.Done()
		if err := userEventGetter.GetEventStart(ctx); err != nil {
			log.Error("event getter exit with error")
		}
	}()

	go func() {
		if err := metrics.Listen(cfg.Metrics.Host, cfg.Metrics.Port); err != nil {
			log.Error("failed to start metrics server", slog.String("error", err.Error()))
		}
	}()
	log.Info("starting application")

	appErrChan := make(chan error, 1)
	go func() {
		appErrChan <- application.Run()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-appErrChan:
			if err != nil {
				log.Error("app failed", slog.String("error", err.Error()))
				exitCode = 1
				return
			}
		case stopSignal := <-stop:
			log.Info("shutting down application", slog.String("signal", stopSignal.String()))
		}
	}
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
