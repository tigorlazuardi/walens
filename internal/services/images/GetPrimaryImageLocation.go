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

// ImageLocationResult contains the essential location info for serving.
type ImageLocationResult struct {
	ImageID dbtypes.UUID
	Path    string
}

// GetPrimaryImageLocation retrieves the primary location record for an image.
func (s *Service) GetPrimaryImageLocation(ctx context.Context, imageID dbtypes.UUID) (*ImageLocationResult, error) {
	var location model.ImageLocations
	stmt := SELECT(ImageLocations.AllColumns).
		FROM(ImageLocations).
		WHERE(ImageLocations.ImageID.EQ(String(imageID.String())).
			AND(ImageLocations.IsActive.EQ(Int(1)))).
		ORDER_BY(ImageLocations.IsPrimary.DESC()).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &location); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, fmt.Errorf("get primary image location: %w", err)
	}
	return &ImageLocationResult{
		ImageID: location.ImageID,
		Path:    location.Path,
	}, nil
}
