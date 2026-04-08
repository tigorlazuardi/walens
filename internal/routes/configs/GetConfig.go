package configs

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	configsvc "github.com/walens/walens/internal/services/configs"
)

type PersistedConfig = configsvc.PersistedConfig

// GetConfigOperation returns the Huma operation metadata for GetConfig.
func GetConfigOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-configs-get-config",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/configs/GetConfig"),
		Summary:     "Get persisted config",
		Description: "Returns the current persisted configuration. Initializes defaults if no config exists. Note: BasePath and Auth settings are bootstrap-only and not included.",
		Tags:        []string{"configs"},
	}
}

// GetConfigInput describes the request body for GetConfig (currently empty).
type GetConfigInput struct {
	Body struct{}
}

// GetConfigOutput describes the response body for GetConfig.
type GetConfigOutput struct {
	Body PersistedConfig
}

// GetConfig handles POST /api/v1/configs/GetConfig.
// Returns the active persisted config, initializing defaults if needed.
func GetConfig(ctx context.Context, input *GetConfigInput, svc *configsvc.Service) (*GetConfigOutput, error) {
	cfg, err := svc.GetConfig(ctx)
	if err != nil {
		if errors.Is(err, configsvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		return nil, huma.Error500InternalServerError("failed to get config", err)
	}

	return &GetConfigOutput{
		Body: *cfg,
	}, nil
}
