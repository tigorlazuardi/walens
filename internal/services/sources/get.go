package sources

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// GetSource returns a single source by ID.
func (s *Service) GetSource(ctx context.Context, id dbtypes.UUID) (*SourceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	var src model.Sources
	stmt := SELECT(Sources.AllColumns).
		FROM(Sources).
		WHERE(Sources.ID.EQ(String(id.UUID.String()))).
		LIMIT(1)

	if err := stmt.QueryContext(ctx, s.db, &src); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrSourceNotFound
		}
		return nil, fmt.Errorf("query source: %w", err)
	}

	return &src, nil
}
