package main

import (
	"LogFlux/services/logs/internal/config"
	"LogFlux/services/logs/internal/repository/clickhouse"
	"errors"
	"flag"
	"log/slog"
	"os"
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

	slog.Info("configuration loaded", "kafka_topic", cfg.Kafka.Topic)

	conn, err := clickhouse.NewClient(ctx, cfg.ClickHouse)
	if err != nil {
		return err
	}
	defer conn.Close()
	return errors.New("Logs service is not implemented yet")
}
