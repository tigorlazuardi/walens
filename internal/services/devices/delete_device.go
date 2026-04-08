package devices

import (
	"context"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

func (s *Service) DeleteDevice(ctx context.Context, id dbtypes.UUID) error {
	if s.db == nil {
		return ErrDBUnavailable
	}
	if _, err := s.GetDevice(ctx, id); err != nil {
		return err
	}
	stmt := Devices.DELETE().WHERE(Devices.ID.EQ(String(id.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	return nil
}
