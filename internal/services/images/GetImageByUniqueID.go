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

// GetImageByUniqueID looks up an image by its source and unique identifier combination.
func (s *Service) GetImageByUniqueID(ctx context.Context, sourceID dbtypes.UUID, uniqueID string) (*model.Images, error) {
	var img model.Images
	stmt := SELECT(Images.AllColumns).
		FROM(Images).
		WHERE(Images.SourceID.EQ(String(sourceID.String())).
			AND(Images.UniqueIdentifier.EQ(String(uniqueID)))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &img); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("get image by unique id: %w", err)
	}
	return &img, nil
}
