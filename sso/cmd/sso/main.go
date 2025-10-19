package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	grpcapp "sso/internal/app/grpc"
	"sso/internal/config"
	kafkaproducer "sso/internal/lib/kafka"
	eventsender "sso/internal/services/event-sender"
	"sso/internal/storage/sqlite"
	"sync"
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
	exitCode := 0

	defer func() {
		os.Exit(exitCode)
	}()

	log.Info("starting application")

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to create storage", slog.String("error", err.Error()))
		exitCode = 1
		return
	}
	kafkaProducer, err := kafkaproducer.New(
		log, cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.DialAdress)
	if err != nil {
		log.Error("failed to create storage", slog.String("error", err.Error()))
		os.Exit(exitCode)
		return
	}
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			log.Error("Kafka close error", slog.String("err", err.Error()))
			exitCode = 1
		}
	}()

	eventSender := eventsender.New(log, storage, kafkaProducer)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	defer func() {
		cancel()
		wg.Wait()
	}()

	go func() {
		defer wg.Done()
		if err := eventSender.StartProcessingEvents(ctx, 5*time.Second); err != nil {
			if errors.Is(err, context.Canceled) {
				log.Info("Event sender stopped")
				return
			}
			log.Error("Event sender stopped with error", slog.String("error", err.Error()))
			exitCode = 1
			return
		}
	}()

	application, err := grpcapp.New(
		log,
		grpcapp.AppConfig{
			GrpcPort:    cfg.GRPC.Port,
			StoragePath: cfg.StoragePath,
			TokenTTL:    cfg.TokenTTL,
		},
	)
	if err != nil {
		log.Error("failed to init app", slog.String("error", err.Error()))
		exitCode = 1
		return
	}

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
			return
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
