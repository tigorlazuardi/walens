package configs

import (
	"context"
	"errors"
)

// ErrDBUnavailable is returned when the database is not available.
var ErrDBUnavailable = errors.New("database unavailable")

// GetConfig returns the current persisted config, initializing defaults if needed.
// If the database is unavailable, returns ErrDBUnavailable.
func (s *Service) GetConfig(ctx context.Context) (*PersistedConfig, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	cfg, err := s.Load(ctx)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, ErrConfigNotFound) {
		return nil, err
	}

	// Config row is absent or empty; inject defaults and store them.
	defaultCfg := DefaultPersistedConfig()
	if err := s.Store(ctx, defaultCfg); err != nil {
		return nil, err
	}

	return defaultCfg, nil
}
