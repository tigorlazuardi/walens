package sources

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type UpdateSourceRequest struct {
	ID          dbtypes.UUID     `json:"id" doc:"Unique source identifier."`
	Name        *string          `json:"name,omitempty" doc:"Unique human-readable source name."`
	SourceType  *string          `json:"source_type,omitempty" doc:"Registered source implementation name."`
	Params      *json.RawMessage `json:"params,omitempty" doc:"Source-specific configuration as JSON."`
	LookupCount *int64           `json:"lookup_count,omitempty" doc:"Upstream lookup budget per run (0 = use source default)."`
	IsEnabled   *bool            `json:"is_enabled,omitempty" doc:"Whether this source is active."`
}

type UpdateSourceResponse = model.Sources

// UpdateSource updates an existing source with full-object update semantics.
func (s *Service) UpdateSource(ctx context.Context, req UpdateSourceRequest) (UpdateSourceResponse, error) {
	existing, err := s.GetSource(ctx, GetSourceRequest{ID: req.ID})
	if err != nil {
		return UpdateSourceResponse{}, err
	}

	mergedName := existing.Name
	if req.Name != nil {
		mergedName = *req.Name
	}
	mergedSourceType := existing.SourceType
	if req.SourceType != nil {
		mergedSourceType = *req.SourceType
	}
	mergedParams := existing.Params
	if req.Params != nil {
		mergedParams = dbtypes.RawJSON(*req.Params)
	}
	mergedLookupCount := existing.LookupCount
	if req.LookupCount != nil {
		mergedLookupCount = *req.LookupCount
	}
	mergedIsEnabled := existing.IsEnabled
	if req.IsEnabled != nil {
		mergedIsEnabled = dbtypes.BoolInt(*req.IsEnabled)
	}

	if err := s.validateSourceType(mergedSourceType, json.RawMessage(mergedParams)); err != nil {
		if errors.Is(err, ErrRegistryUnavailable) {
			return UpdateSourceResponse{}, huma.Error503ServiceUnavailable("source registry unavailable", err)
		}
		if errors.Is(err, ErrInvalidSourceType) {
			return UpdateSourceResponse{}, huma.Error400BadRequest("invalid source type: not registered", err)
		}
		if errors.Is(err, ErrInvalidParams) {
			return UpdateSourceResponse{}, huma.Error400BadRequest("invalid params for source type", err)
		}
		return UpdateSourceResponse{}, huma.Error500InternalServerError("failed to validate source type", err)
	}

	duplicateCount, err := s.countSources(ctx,
		Sources.Name.EQ(String(mergedName)).
			AND(Sources.ID.NOT_EQ(String(req.ID.UUID.String()))),
	)
	if err != nil {
		return UpdateSourceResponse{}, huma.Error500InternalServerError("failed to check duplicate source name", err)
	}
	if duplicateCount > 0 {
		return UpdateSourceResponse{}, huma.Error409Conflict("source with this name already exists", ErrDuplicateSourceName)
	}

	updated := existing
	updated.Name = mergedName
	updated.SourceType = mergedSourceType
	updated.Params = mergedParams
	updated.LookupCount = mergedLookupCount
	updated.IsEnabled = mergedIsEnabled
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()

	stmt := Sources.UPDATE(
		Sources.Name,
		Sources.SourceType,
		Sources.Params,
		Sources.LookupCount,
		Sources.IsEnabled,
		Sources.UpdatedAt,
	).MODEL(updated).WHERE(
		Sources.ID.EQ(String(req.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return UpdateSourceResponse{}, huma.Error500InternalServerError("failed to update source", err)
	}

	// Trigger scheduler reload if enabled state changed
	if req.IsEnabled != nil && s.scheduler != nil {
		_ = s.scheduler.Reload()
	}

	return s.GetSource(ctx, GetSourceRequest{ID: req.ID})
}
