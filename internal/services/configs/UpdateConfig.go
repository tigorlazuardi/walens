package configs

import (
	"context"
)

// UpdateConfig atomically replaces the entire persisted config.
// Returns the newly stored config on success.
// If the database is unavailable, returns ErrDBUnavailable.
func (s *Service) UpdateConfig(ctx context.Context, cfg *PersistedConfig) (*PersistedConfig, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	if err := s.Store(ctx, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
