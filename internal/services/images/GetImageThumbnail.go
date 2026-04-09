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

// GetImageThumbnail retrieves the thumbnail record for a given image.
func (s *Service) GetImageThumbnail(ctx context.Context, imageID dbtypes.UUID) (*model.ImageThumbnails, error) {
	var thumbnail model.ImageThumbnails
	stmt := SELECT(ImageThumbnails.AllColumns).
		FROM(ImageThumbnails).
		WHERE(ImageThumbnails.ImageID.EQ(String(imageID.String()))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &thumbnail); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrThumbnailNotFound
		}
		return nil, fmt.Errorf("get image thumbnail: %w", err)
	}
	return &thumbnail, nil
}
