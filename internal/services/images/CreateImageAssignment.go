package images

import (
	"context"
	"fmt"

	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// CreateImageAssignment creates a new assignment of an image to a device.
func (s *Service) CreateImageAssignment(ctx context.Context, imageID, deviceID dbtypes.UUID) (*model.ImageAssignments, error) {
	now := dbtypes.NewUnixMilliTimeNow()
	id := dbtypes.MustNewUUIDV7()

	row := model.ImageAssignments{
		ID:        id,
		ImageID:   imageID,
		DeviceID:  deviceID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	stmt := ImageAssignments.INSERT(
		ImageAssignments.ID, ImageAssignments.ImageID,
		ImageAssignments.DeviceID, ImageAssignments.CreatedAt, ImageAssignments.UpdatedAt,
	).MODEL(row)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("create image assignment: %w", err)
	}

	return &row, nil
}
