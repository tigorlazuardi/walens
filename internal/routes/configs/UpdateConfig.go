package configs

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	configsvc "github.com/walens/walens/internal/services/configs"
)

type UpdateConfigBody = configsvc.PersistedConfig

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
	Body UpdateConfigBody
}

// UpdateConfigOutput describes the response body for UpdateConfig.
type UpdateConfigOutput struct {
	Body UpdateConfigBody
}

// UpdateConfig handles POST /api/v1/configs/UpdateConfig.
// Atomically replaces the entire persisted config and returns the new values.
// Note: BasePath and Auth settings are bootstrap-only and not affected by this operation.

func UpdateConfig(ctx context.Context, input *UpdateConfigInput, svc *configsvc.Service) (*UpdateConfigOutput, error) {
	cfg := &input.Body
	storedCfg, err := svc.UpdateConfig(ctx, cfg)
	if err != nil {
		if errors.Is(err, configsvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		return nil, huma.Error500InternalServerError("failed to update config", err)
	}

	return &UpdateConfigOutput{
		Body: *storedCfg,
	}, nil
}
