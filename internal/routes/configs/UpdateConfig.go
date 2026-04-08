package configs

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/services/configs"
)

// UpdateConfigOperation returns the Huma operation metadata for UpdateConfig.
func UpdateConfigOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-configs-update-config",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/configs/UpdateConfig"),
		Summary:     "Update persisted config",
		Description: "Atomically replaces the entire persisted configuration. Note: BasePath and Auth settings are bootstrap-only and cannot be changed via this endpoint.",
		Tags:        []string{"configs"},
	}
}

// UpdateConfigInput describes the request body for UpdateConfig.
type UpdateConfigInput struct {
	Body struct {
		// Directory for storing application data.
		DataDir string `json:"data_dir" doc:"Directory for storing application data."`
		// Logging level (debug, info, warn, error).
		LogLevel string `json:"log_level" doc:"Logging level (debug, info, warn, error)."`
	}
}

// UpdateConfigOutput describes the response body for UpdateConfig.
type UpdateConfigOutput struct {
	Body struct {
		// The newly stored persisted configuration values.
		DataDir  string `json:"data_dir" doc:"Directory for storing application data."`
		LogLevel string `json:"log_level" doc:"Logging level (debug, info, warn, error)."`
	}
}

// UpdateConfig handles POST /api/v1/configs/UpdateConfig.
// Atomically replaces the entire persisted config and returns the new values.
// Note: BasePath and Auth settings are bootstrap-only and not affected by this operation.
func UpdateConfig(ctx context.Context, input *UpdateConfigInput, svc *configs.Service) (*UpdateConfigOutput, error) {
	cfg := &configs.PersistedConfig{
		DataDir:  input.Body.DataDir,
		LogLevel: input.Body.LogLevel,
	}

	storedCfg, err := svc.UpdateConfig(ctx, cfg)
	if err != nil {
		if errors.Is(err, configs.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		return nil, huma.Error500InternalServerError("failed to update config", err)
	}

	return &UpdateConfigOutput{
		Body: struct {
			DataDir  string `json:"data_dir" doc:"Directory for storing application data."`
			LogLevel string `json:"log_level" doc:"Logging level (debug, info, warn, error)."`
		}{
			DataDir:  storedCfg.DataDir,
			LogLevel: storedCfg.LogLevel,
		},
	}, nil
}
