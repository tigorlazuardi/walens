package sources

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
)

// ListSources returns all configured source rows.
func (s *Service) ListSources(ctx context.Context) ([]SourceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	var results []model.Sources
	stmt := SELECT(Sources.AllColumns).
		FROM(Sources).
		ORDER_BY(Sources.Name.ASC())

	if err := stmt.QueryContext(ctx, s.db, &results); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return []SourceRow{}, nil
		}
		return nil, fmt.Errorf("query sources: %w", err)
	}

	return results, nil
}
