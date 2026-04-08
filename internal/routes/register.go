package routes

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	configsroutes "github.com/walens/walens/internal/routes/configs"
	configssvc "github.com/walens/walens/internal/services/configs"
)

// RegisterConfigsRoutes registers all configs RPC routes under /api/v1/configs/.
func RegisterConfigsRoutes(api huma.API, basePath string, configService *configssvc.Service) {
	if configService == nil {
		configService = configssvc.NewService(nil)
	}

	huma.Register(api, configsroutes.GetConfigOperation(basePath), func(ctx context.Context, input *configsroutes.GetConfigInput) (*configsroutes.GetConfigOutput, error) {
		return configsroutes.GetConfig(ctx, input, configService)
	})

	huma.Register(api, configsroutes.UpdateConfigOperation(basePath), func(ctx context.Context, input *configsroutes.UpdateConfigInput) (*configsroutes.UpdateConfigOutput, error) {
		return configsroutes.UpdateConfig(ctx, input, configService)
	})
}
