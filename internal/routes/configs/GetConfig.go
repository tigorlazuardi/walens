package configs

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/services/configs"
)

// GetConfigInput describes the request body for GetConfig (currently empty).
type GetConfigInput struct {
	Body struct{}
}

// GetConfigOutput describes the response body for GetConfig.
type GetConfigOutput struct {
	Body struct {
		// Persisted configuration values.
		DataDir  string `json:"data_dir" doc:"Directory for storing application data."`
		LogLevel string `json:"log_level" doc:"Logging level (debug, info, warn, error)."`
	}
}

// GetConfig handles POST /api/v1/configs/GetConfig.
// Returns the active persisted config, initializing defaults if needed.
func GetConfig(ctx context.Context, input *GetConfigInput, svc *configs.Service) (*GetConfigOutput, error) {
	cfg, err := svc.GetConfig(ctx)
	if err != nil {
		if errors.Is(err, configs.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		return nil, huma.Error500InternalServerError("failed to get config", err)
	}

	return &GetConfigOutput{
		Body: struct {
			DataDir  string `json:"data_dir" doc:"Directory for storing application data."`
			LogLevel string `json:"log_level" doc:"Logging level (debug, info, warn, error)."`
		}{
			DataDir:  cfg.DataDir,
			LogLevel: cfg.LogLevel,
		},
	}, nil
}
