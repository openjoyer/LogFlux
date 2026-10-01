package main

import (
	"LogFlux/services/ingestion/internal/config"
	"LogFlux/services/ingestion/internal/middleware"
	"LogFlux/services/ingestion/internal/producer"
	"LogFlux/services/ingestion/internal/ratelimit"
	"LogFlux/services/ingestion/internal/service"
	httptransport "LogFlux/services/ingestion/internal/transport/http"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("service", "ingestion")

	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String(
		"config",
		"services/ingestion/config/local.yaml",
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

	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Address,
		DB:           cfg.Redis.DB,
		Password:     cfg.Redis.Password,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	defer redisClient.Close()

	pingCtx, cancel := context.WithTimeout(ctx, cfg.Redis.PingTimeout)
	err = redisClient.Ping(pingCtx).Err()
	cancel()
	if err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}

	publisher := producer.NewKafkaProducer(&cfg.Kafka)
	defer publisher.Close()

	ingestionService := service.NewIngestionService(publisher)
	handler := httptransport.NewHandler(ingestionService)

	limiter := ratelimit.NewLimiter(redisClient, cfg.RateLimit.Limit)

	common := func(next http.Handler) http.Handler {
		return middleware.Chain(
			next,
			middleware.Logging,
			middleware.RequestID,
			middleware.Metrics,
		)
	}

	protected := func(next http.Handler) http.Handler {
		return middleware.Chain(
			next,
			middleware.Authentication,
			middleware.RateLimiting(limiter),
		)
	}

	router := httptransport.NewRouter(handler, common, protected)

	server := &http.Server{
		Addr:    net.JoinHostPort(cfg.HTTP.Address, cfg.HTTP.Port),
		Handler: router,
	}

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown HTTP: %w", err)
		}
		return nil
	}
}
