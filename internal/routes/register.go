package routes

import (
	"context"
	"net/http"
	"path"

	"github.com/danielgtaylor/huma/v2"
	configsroutes "github.com/walens/walens/internal/routes/configs"
	configssvc "github.com/walens/walens/internal/services/configs"
)

// RegisterConfigsRoutes registers all configs RPC routes under /api/v1/configs/.
func RegisterConfigsRoutes(api huma.API, basePath string, configService *configssvc.Service) {
	if configService == nil {
		configService = configssvc.NewService(nil)
	}

	prefix := path.Join(basePath, "/api/v1/configs")

	huma.Register(api, huma.Operation{
		OperationID: "post-configs-get-config",
		Method:      http.MethodPost,
		Path:        path.Join(prefix, "GetConfig"),
		Summary:     "Get persisted config",
		Description: "Returns the current persisted configuration. Initializes defaults if no config exists. Note: BasePath and Auth settings are bootstrap-only and not included.",
		Tags:        []string{"configs"},
	}, func(ctx context.Context, input *configsroutes.GetConfigInput) (*configsroutes.GetConfigOutput, error) {
		return configsroutes.GetConfig(ctx, input, configService)
	})

	huma.Register(api, huma.Operation{
		OperationID: "post-configs-update-config",
		Method:      http.MethodPost,
		Path:        path.Join(prefix, "UpdateConfig"),
		Summary:     "Update persisted config",
		Description: "Atomically replaces the entire persisted configuration. Note: BasePath and Auth settings are bootstrap-only and cannot be changed via this endpoint.",
		Tags:        []string{"configs"},
	}, func(ctx context.Context, input *configsroutes.UpdateConfigInput) (*configsroutes.UpdateConfigOutput, error) {
		return configsroutes.UpdateConfig(ctx, input, configService)
	})
}
