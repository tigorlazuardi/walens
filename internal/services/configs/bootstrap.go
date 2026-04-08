package configs

import (
	"context"
	"errors"
	"fmt"
)

// BootstrapDefault loads the persisted config, or if absent/empty, inserts
// the provided default config and returns it. This ensures the app always
// has a valid persisted config after bootstrap.
func (s *Service) BootstrapDefault(ctx context.Context, defaultCfg *PersistedConfig) (*PersistedConfig, error) {
	existing, err := s.Load(ctx)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrConfigNotFound) {
		return nil, err
	}

	// Config row is absent or empty; inject defaults via atomic insert.
	if err := s.Store(ctx, defaultCfg); err != nil {
		return nil, fmt.Errorf("bootstrap default config: %w", err)
	}

	return defaultCfg, nil
}
