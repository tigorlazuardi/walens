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

type CreateSourceRequest struct {
	Name        string          `json:"name" doc:"Unique human-readable source name."`
	SourceType  string          `json:"source_type" doc:"Registered source implementation name (e.g., booru, reddit)."`
	Params      json.RawMessage `json:"params" doc:"Source-specific configuration as JSON."`
	LookupCount int64           `json:"lookup_count" doc:"Upstream lookup budget per run (0 = use source default)."`
	IsEnabled   bool            `json:"is_enabled" doc:"Whether this source is active."`
}

type CreateSourceResponse = model.Sources

// CreateSource creates a new source row.
func (s *Service) CreateSource(ctx context.Context, req CreateSourceRequest) (CreateSourceResponse, error) {
	if err := s.validateSourceType(req.SourceType, req.Params); err != nil {
		if errors.Is(err, ErrRegistryUnavailable) {
			return CreateSourceResponse{}, huma.Error503ServiceUnavailable("source registry unavailable", err)
		}
		if errors.Is(err, ErrInvalidSourceType) {
			return CreateSourceResponse{}, huma.Error400BadRequest("invalid source type: not registered", err)
		}
		if errors.Is(err, ErrInvalidParams) {
			return CreateSourceResponse{}, huma.Error400BadRequest("invalid params for source type", err)
		}
		return CreateSourceResponse{}, huma.Error500InternalServerError("failed to validate source type", err)
	}

	duplicateCount, err := s.countSources(ctx, Sources.Name.EQ(String(req.Name)))
	if err != nil {
		return CreateSourceResponse{}, huma.Error500InternalServerError("failed to check duplicate source name", err)
	}
	if duplicateCount > 0 {
		return CreateSourceResponse{}, huma.Error409Conflict("source with this name already exists", ErrDuplicateSourceName)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return CreateSourceResponse{}, huma.Error500InternalServerError("failed to generate source id", err)
	}

	row := model.Sources{
		ID:          &id,
		Name:        req.Name,
		SourceType:  req.SourceType,
		Params:      dbtypes.RawJSON(req.Params),
		LookupCount: req.LookupCount,
		IsEnabled:   dbtypes.BoolInt(req.IsEnabled),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	stmt := Sources.INSERT(
		Sources.ID,
		Sources.Name,
		Sources.SourceType,
		Sources.Params,
		Sources.LookupCount,
		Sources.IsEnabled,
		Sources.CreatedAt,
		Sources.UpdatedAt,
	).MODEL(row)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return CreateSourceResponse{}, huma.Error500InternalServerError("failed to create source", err)
	}

	return s.GetSource(ctx, GetSourceRequest{ID: id})
}
