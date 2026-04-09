package configs

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type UpdateConfigRequest struct {
	DataDir  string `json:"data_dir" doc:"Directory for storing application data."`
	LogLevel string `json:"log_level" doc:"Logging level (debug, info, warn, error)."`
}

type UpdateConfigResponse = PersistedConfig

// UpdateConfig atomically replaces the entire persisted config.
// Returns the newly stored config on success.
func (s *Service) UpdateConfig(ctx context.Context, req UpdateConfigRequest) (UpdateConfigResponse, error) {
	cfg := &PersistedConfig{DataDir: req.DataDir, LogLevel: req.LogLevel}
	if err := s.Store(ctx, cfg); err != nil {
		return UpdateConfigResponse{}, huma.Error500InternalServerError("failed to update config", err)
	}

	return *cfg, nil
}
