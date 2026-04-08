package sources

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type UpdateSourceInput struct {
	ID          dbtypes.UUID     `json:"id" doc:"Unique source identifier."`
	Name        *string          `json:"name,omitempty" doc:"Unique human-readable source name."`
	SourceType  *string          `json:"source_type,omitempty" doc:"Registered source implementation name."`
	Params      *json.RawMessage `json:"params,omitempty" doc:"Source-specific configuration as JSON."`
	LookupCount *int64           `json:"lookup_count,omitempty" doc:"Upstream lookup budget per run (0 = use source default)."`
	IsEnabled   *bool            `json:"is_enabled,omitempty" doc:"Whether this source is active."`
}

// UpdateSource updates an existing source with full-object update semantics.
func (s *Service) UpdateSource(ctx context.Context, input *UpdateSourceInput) (*SourceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	existing, err := s.GetSource(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	mergedName := existing.Name
	if input.Name != nil {
		mergedName = *input.Name
	}
	mergedSourceType := existing.SourceType
	if input.SourceType != nil {
		mergedSourceType = *input.SourceType
	}
	mergedParams := existing.Params
	if input.Params != nil {
		mergedParams = dbtypes.RawJSON(*input.Params)
	}
	mergedLookupCount := existing.LookupCount
	if input.LookupCount != nil {
		mergedLookupCount = *input.LookupCount
	}
	mergedIsEnabled := existing.IsEnabled
	if input.IsEnabled != nil {
		mergedIsEnabled = dbtypes.BoolInt(*input.IsEnabled)
	}

	if err := s.validateSourceType(mergedSourceType, json.RawMessage(mergedParams)); err != nil {
		return nil, err
	}

	duplicateCount, err := s.countSources(ctx,
		Sources.Name.EQ(String(mergedName)).
			AND(Sources.ID.NOT_EQ(String(input.ID.UUID.String()))),
	)
	if err != nil {
		return nil, fmt.Errorf("check duplicate name: %w", err)
	}
	if duplicateCount > 0 {
		return nil, ErrDuplicateSourceName
	}

	updated := *existing
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
		Sources.ID.EQ(String(input.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("update source: %w", err)
	}

	return s.GetSource(ctx, input.ID)
}
