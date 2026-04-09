package images

import (
	"context"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// GetImageLocations retrieves all location records for a given image.
func (s *Service) GetImageLocations(ctx context.Context, imageID dbtypes.UUID) ([]model.ImageLocations, error) {
	var locations []model.ImageLocations
	stmt := SELECT(ImageLocations.AllColumns).
		FROM(ImageLocations).
		WHERE(ImageLocations.ImageID.EQ(String(imageID.String())))
	if err := stmt.QueryContext(ctx, s.db, &locations); err != nil {
		return nil, fmt.Errorf("get image locations: %w", err)
	}
	return locations, nil
}
