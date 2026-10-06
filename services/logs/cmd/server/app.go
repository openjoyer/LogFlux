package main

import (
	"LogFlux/services/logs/cmd/di"
	"LogFlux/services/logs/internal/config"
	"LogFlux/services/logs/internal/transport/kafka"
	"context"
)

type App struct {
	container *di.Container
	pipeline  *kafka.Pipeline
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	container, err := di.NewContainer(ctx, cfg)
	if err != nil {
		return nil, err
	}

	pipeline := container.Pipeline()

	return &App{
		container: container,
		pipeline:  pipeline,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	return a.pipeline.Run(ctx)
}

func (a *App) Close() error {
	return a.container.Close()
}
