package routes

import (
	"context"
	"database/sql"

	"github.com/danielgtaylor/huma/v2"
	configsroutes "github.com/walens/walens/internal/routes/configs"
	schedulesroutes "github.com/walens/walens/internal/routes/source_schedules"
	sourcetypesroutes "github.com/walens/walens/internal/routes/source_types"
	sourceroutes "github.com/walens/walens/internal/routes/sources"
	"github.com/walens/walens/internal/scheduler"
	configssvc "github.com/walens/walens/internal/services/configs"
	schedulessvc "github.com/walens/walens/internal/services/source_schedules"
	sourcetypessvc "github.com/walens/walens/internal/services/source_types"
	sourcessvc "github.com/walens/walens/internal/services/sources"
	"github.com/walens/walens/internal/sources"
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

// RegisterSourceTypesRoutes registers all source_types RPC routes under /api/v1/source_types/.
func RegisterSourceTypesRoutes(api huma.API, basePath string, registry *sources.Registry) {
	sourceTypesService := sourcetypessvc.NewService(registry)

	huma.Register(api, sourcetypesroutes.ListSourceTypesOperation(basePath), func(ctx context.Context, input *sourcetypesroutes.ListSourceTypesInput) (*sourcetypesroutes.ListSourceTypesOutput, error) {
		return sourcetypesroutes.ListSourceTypes(ctx, input, sourceTypesService)
	})

	huma.Register(api, sourcetypesroutes.GetSourceTypeOperation(basePath), func(ctx context.Context, input *sourcetypesroutes.GetSourceTypeInput) (*sourcetypesroutes.GetSourceTypeOutput, error) {
		return sourcetypesroutes.GetSourceType(ctx, input, sourceTypesService)
	})
}

// RegisterSourcesRoutes registers all sources RPC routes under /api/v1/sources/.
func RegisterSourcesRoutes(api huma.API, basePath string, dbSourcesService *sourcessvc.Service) {
	if dbSourcesService == nil {
		dbSourcesService = sourcessvc.NewService(nil, nil)
	}

	huma.Register(api, sourceroutes.ListSourcesOperation(basePath), func(ctx context.Context, input *sourceroutes.ListSourcesInput) (*sourceroutes.ListSourcesOutput, error) {
		return sourceroutes.ListSources(ctx, input, dbSourcesService)
	})

	huma.Register(api, sourceroutes.GetSourceOperation(basePath), func(ctx context.Context, input *sourceroutes.GetSourceInput) (*sourceroutes.GetSourceOutput, error) {
		return sourceroutes.GetSource(ctx, input, dbSourcesService)
	})

	huma.Register(api, sourceroutes.CreateSourceOperation(basePath), func(ctx context.Context, input *sourceroutes.CreateSourceInput) (*sourceroutes.CreateSourceOutput, error) {
		return sourceroutes.CreateSource(ctx, input, dbSourcesService)
	})

	huma.Register(api, sourceroutes.UpdateSourceOperation(basePath), func(ctx context.Context, input *sourceroutes.UpdateSourceInput) (*sourceroutes.UpdateSourceOutput, error) {
		return sourceroutes.UpdateSource(ctx, input, dbSourcesService)
	})

	huma.Register(api, sourceroutes.DeleteSourceOperation(basePath), func(ctx context.Context, input *sourceroutes.DeleteSourceInput) (*struct{}, error) {
		return sourceroutes.DeleteSource(ctx, input, dbSourcesService)
	})
}

// RegisterSourceSchedulesRoutes registers all source_schedules RPC routes under /api/v1/source_schedules/.
func RegisterSourceSchedulesRoutes(api huma.API, basePath string, db *sql.DB, sc *scheduler.Scheduler) {
	var schedService *schedulessvc.Service
	if db != nil {
		schedService = schedulessvc.NewService(db, sc)
	} else {
		schedService = schedulessvc.NewService(nil, nil)
	}

	huma.Register(api, schedulesroutes.ListSourceSchedulesOperation(basePath), func(ctx context.Context, input *schedulesroutes.ListSourceSchedulesInput) (*schedulesroutes.ListSourceSchedulesOutput, error) {
		return schedulesroutes.ListSourceSchedules(ctx, input, schedService)
	})

	huma.Register(api, schedulesroutes.GetSourceScheduleOperation(basePath), func(ctx context.Context, input *schedulesroutes.GetSourceScheduleInput) (*schedulesroutes.GetSourceScheduleOutput, error) {
		return schedulesroutes.GetSourceSchedule(ctx, input, schedService)
	})

	huma.Register(api, schedulesroutes.CreateSourceScheduleOperation(basePath), func(ctx context.Context, input *schedulesroutes.CreateSourceScheduleInput) (*schedulesroutes.CreateSourceScheduleOutput, error) {
		return schedulesroutes.CreateSourceSchedule(ctx, input, schedService)
	})

	huma.Register(api, schedulesroutes.UpdateSourceScheduleOperation(basePath), func(ctx context.Context, input *schedulesroutes.UpdateSourceScheduleInput) (*schedulesroutes.UpdateSourceScheduleOutput, error) {
		return schedulesroutes.UpdateSourceSchedule(ctx, input, schedService)
	})

	huma.Register(api, schedulesroutes.DeleteSourceScheduleOperation(basePath), func(ctx context.Context, input *schedulesroutes.DeleteSourceScheduleInput) (*struct{}, error) {
		return schedulesroutes.DeleteSourceSchedule(ctx, input, schedService)
	})
}
