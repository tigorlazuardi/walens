package sources

import (
	"context"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// DeleteSource deletes a source by ID.
func (s *Service) DeleteSource(ctx context.Context, id dbtypes.UUID) error {
	if s.db == nil {
		return ErrDBUnavailable
	}

	if _, err := s.GetSource(ctx, id); err != nil {
		return err
	}

	stmt := Sources.DELETE().WHERE(Sources.ID.EQ(String(id.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return fmt.Errorf("delete source: %w", err)
	}

	return nil
}
