package main

import (
	"LogFlux/services/logs/internal/config"
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("service", "logs")

	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String(
		"config",
		"services/logs/config/local.yaml",
		"path to configuration",
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	application, err := New(ctx, cfg)
	if err != nil {
		return err
	}

	runErr := application.Run(ctx)

	closeErr := application.Close()

	if errors.Is(runErr, context.Canceled) && ctx.Err() != nil {
		runErr = nil
	}

	return errors.Join(runErr, closeErr)
}
