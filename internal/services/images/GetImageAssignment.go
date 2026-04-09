package images

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

// GetImageAssignment checks if an image is assigned to a specific device.
func (s *Service) GetImageAssignment(ctx context.Context, imageID, deviceID dbtypes.UUID) (*model.ImageAssignments, error) {
	var assignment model.ImageAssignments
	stmt := SELECT(ImageAssignments.AllColumns).
		FROM(ImageAssignments).
		WHERE(ImageAssignments.ImageID.EQ(String(imageID.String())).
			AND(ImageAssignments.DeviceID.EQ(String(deviceID.String())))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &assignment); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrAssignmentNotFound
		}
		return nil, fmt.Errorf("get image assignment: %w", err)
	}
	return &assignment, nil
}
