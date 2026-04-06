package runner

import (
	"context"
	"log/slog"
	"sync"
)

type Runner struct {
	logger *slog.Logger
	mu     sync.RWMutex
}

func New(logger *slog.Logger) *Runner {
	return &Runner{
		logger: logger,
	}
}

func (r *Runner) Start(ctx context.Context) error {
	r.logger.Info("runner starting")
	return nil
}

func (r *Runner) Stop() error {
	r.logger.Info("runner stopping")
	return nil
}

func (r *Runner) Run(ctx context.Context) error {
	return nil
}
