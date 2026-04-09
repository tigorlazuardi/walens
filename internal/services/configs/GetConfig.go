package configs

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
)

type GetConfigRequest struct{}

type GetConfigResponse = PersistedConfig

// GetConfig returns the current persisted config, initializing defaults if needed.
func (s *Service) GetConfig(ctx context.Context, _ GetConfigRequest) (GetConfigResponse, error) {
	cfg, err := s.Load(ctx)
	if err == nil {
		return *cfg, nil
	}
	if !errors.Is(err, ErrConfigNotFound) {
		return GetConfigResponse{}, huma.Error500InternalServerError("failed to get config", err)
	}

	// Config row is absent or empty; inject defaults and store them.
	defaultCfg := DefaultPersistedConfig()
	if err := s.Store(ctx, defaultCfg); err != nil {
		return GetConfigResponse{}, huma.Error500InternalServerError("failed to initialize config defaults", err)
	}

	return *defaultCfg, nil
}
