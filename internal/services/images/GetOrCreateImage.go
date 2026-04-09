package images

import (
	"context"
	"errors"
	"fmt"

	"github.com/walens/walens/internal/db/generated/model"
)

// GetOrCreateImage attempts to find an existing image by source+uniqueID, or creates a new one if not found.
// Returns the image, a boolean indicating if it was newly created, and any error.
func (s *Service) GetOrCreateImage(ctx context.Context, req CreateImageRequest) (*model.Images, bool, error) {
	existing, err := s.GetImageByUniqueID(ctx, req.SourceID, req.UniqueIdentifier)
	if err == nil {
		return existing, false, nil
	}
	if !errors.Is(err, ErrImageNotFound) {
		return nil, false, fmt.Errorf("get image by unique id: %w", err)
	}

	created, err := s.CreateImage(ctx, req)
	if err != nil {
		return nil, false, fmt.Errorf("create image: %w", err)
	}

	return created, true, nil
}
