package sources

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type CreateSourceInput struct {
	Name        string          `json:"name" doc:"Unique human-readable source name."`
	SourceType  string          `json:"source_type" doc:"Registered source implementation name (e.g., booru, reddit)."`
	Params      json.RawMessage `json:"params" doc:"Source-specific configuration as JSON."`
	LookupCount int64           `json:"lookup_count" doc:"Upstream lookup budget per run (0 = use source default)."`
	IsEnabled   bool            `json:"is_enabled" doc:"Whether this source is active."`
}

// CreateSource creates a new source row.
func (s *Service) CreateSource(ctx context.Context, input *CreateSourceInput) (*SourceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	if err := s.validateSourceType(input.SourceType, input.Params); err != nil {
		return nil, err
	}

	duplicateCount, err := s.countSources(ctx, Sources.Name.EQ(String(input.Name)))
	if err != nil {
		return nil, fmt.Errorf("check duplicate name: %w", err)
	}
	if duplicateCount > 0 {
		return nil, ErrDuplicateSourceName
	}

	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return nil, fmt.Errorf("generate UUIDv7: %w", err)
	}

	row := model.Sources{
		ID:          &id,
		Name:        input.Name,
		SourceType:  input.SourceType,
		Params:      dbtypes.RawJSON(input.Params),
		LookupCount: input.LookupCount,
		IsEnabled:   dbtypes.BoolInt(input.IsEnabled),
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
		return nil, fmt.Errorf("insert source: %w", err)
	}

	return s.GetSource(ctx, id)
}
