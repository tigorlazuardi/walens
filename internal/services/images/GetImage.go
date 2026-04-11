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

// GetImageInput defines the input for GetImage.
type GetImageInput struct {
	ID dbtypes.UUID `json:"id" required:"true" doc:"Image ID"`
}

// GetImage retrieves an image by its ID.
func (s *Service) GetImage(ctx context.Context, input GetImageInput) (*model.Images, error) {
	var img model.Images
	stmt := SELECT(Images.AllColumns).
		FROM(Images).
		WHERE(Images.ID.EQ(String(input.ID.UUID.String()))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &img); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("get image: %w", err)
	}
	return &img, nil
}
