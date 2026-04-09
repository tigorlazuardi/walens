package images

import (
	"context"
	"errors"
	"fmt"

	"github.com/walens/walens/internal/db/generated/model"
	"github.com/walens/walens/internal/dbtypes"
)

// EnsureImageAssignment returns the existing assignment if it already exists,
// otherwise creates a new assignment. This makes assignment creation idempotent.
func (s *Service) EnsureImageAssignment(ctx context.Context, imageID, deviceID dbtypes.UUID) (*model.ImageAssignments, error) {
	// First check if assignment already exists
	existing, err := s.GetImageAssignment(ctx, imageID, deviceID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrAssignmentNotFound) {
		return nil, fmt.Errorf("check existing assignment: %w", err)
	}

	// Assignment doesn't exist, create it
	return s.CreateImageAssignment(ctx, imageID, deviceID)
}
