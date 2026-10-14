package main

import (
	projectsv1 "LogFlux/contracts/gen/go/projects/v1"
	"LogFlux/services/projects/cmd/di"
	"LogFlux/services/projects/internal/config"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type App struct {
	container       *di.Container
	httpServer      *http.Server
	grpcServer      *grpc.Server
	grpcAddress     string
	shutdownTimeout time.Duration
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	container, err := di.NewContainer(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     cfg.Grpc.Keepalive.MaxConnIdle,
			MaxConnectionAge:      cfg.Grpc.Keepalive.MaxConnAge,
			MaxConnectionAgeGrace: cfg.Grpc.Keepalive.MaxConnAgeGrace,
			Time:                  cfg.Grpc.Keepalive.Time,
			Timeout:               cfg.Grpc.Keepalive.Timeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             cfg.Grpc.Keepalive.MinPingInterval,
			PermitWithoutStream: true,
		}),
	)

	projectsv1.RegisterProjectAuthServiceServer(
		grpcServer,
		container.GRPCHandler(),
	)

	return &App{
		container: container,
		httpServer: &http.Server{
			Addr:              cfg.HTTP.Address,
			Handler:           container.HTTPHandler(),
			ReadHeaderTimeout: 5 * time.Second,
		},
		grpcServer:      grpcServer,
		grpcAddress:     cfg.Grpc.Address,
		shutdownTimeout: cfg.ShutdownTimeout,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	grpcListener, err := net.Listen("tcp", a.grpcAddress)
	if err != nil {
		return fmt.Errorf("listen gRPC: %w", err)
	}

	serverErrors := make(chan error, 2)

	go func() {
		err := a.httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		if err != nil {
			err = fmt.Errorf("serve HTTP: %w", err)
		}
		serverErrors <- err
	}()

	go func() {
		err := a.grpcServer.Serve(grpcListener)
		if errors.Is(err, grpc.ErrServerStopped) {
			err = nil
		}
		if err != nil {
			err = fmt.Errorf("serve gRPC: %w", err)
		}
		serverErrors <- err
	}()

	var runErr error
	received := 0

	select {
	case <-ctx.Done():
	case runErr = <-serverErrors:
		received = 1
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		a.shutdownTimeout,
	)
	defer cancel()

	httpStopped := make(chan error, 1)
	go func() {
		err := a.httpServer.Shutdown(shutdownCtx)
		if err != nil {
			_ = a.httpServer.Close()
		}
		httpStopped <- err
	}()

	grpcStopped := make(chan struct{})
	go func() {
		a.grpcServer.GracefulStop()
		close(grpcStopped)
	}()

	var shutdownErr error
	select {
	case <-grpcStopped:
	case <-shutdownCtx.Done():
		a.grpcServer.Stop()
		<-grpcStopped
		shutdownErr = fmt.Errorf("stop gRPC: %w", shutdownCtx.Err())
	}

	shutdownErr = errors.Join(shutdownErr, <-httpStopped)

	for ; received < 2; received++ {
		runErr = errors.Join(runErr, <-serverErrors)
	}

	return errors.Join(runErr, shutdownErr)
}

func (a *App) Close() error {
	return a.container.Close()
}
