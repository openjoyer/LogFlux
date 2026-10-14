package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"LogFlux/services/projects/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("service", "projects")

	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String(
		"config",
		"services/projects/config/config.yaml",
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
