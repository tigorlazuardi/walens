package scheduler

import (
	"context"
	"log/slog"
	"sync"
)

type Scheduler struct {
	logger *slog.Logger
	mu     sync.RWMutex
}

func New(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		logger: logger,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("scheduler starting")
	return nil
}

func (s *Scheduler) Stop() error {
	s.logger.Info("scheduler stopping")
	return nil
}

func (s *Scheduler) Reload() error {
	s.logger.Info("scheduler reloading")
	return nil
}
