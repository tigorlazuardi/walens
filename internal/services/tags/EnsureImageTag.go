package tags

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

// EnsureImageTag creates an image-tag association if it doesn't already exist.
// The association is idempotent - calling it multiple times with the same imageID
// and tagID will not create duplicate records.
func (s *Service) EnsureImageTag(ctx context.Context, imageID, tagID dbtypes.UUID) (*model.ImageTags, error) {
	// First check if the association already exists
	existing, err := s.getImageTag(ctx, imageID, tagID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrImageTagNotFound) {
		return nil, fmt.Errorf("check existing image tag: %w", err)
	}

	// Association doesn't exist, create it
	return s.createImageTag(ctx, imageID, tagID)
}

// ErrImageTagNotFound is returned when an image-tag association doesn't exist.
var ErrImageTagNotFound = errors.New("image tag not found")

// getImageTag looks up an image-tag association.
func (s *Service) getImageTag(ctx context.Context, imageID, tagID dbtypes.UUID) (*model.ImageTags, error) {
	stmt := SELECT(
		ImageTags.ID, ImageTags.ImageID, ImageTags.TagID, ImageTags.CreatedAt,
	).FROM(
		ImageTags,
	).WHERE(
		ImageTags.ImageID.EQ(String(imageID.UUID.String())).
			AND(ImageTags.TagID.EQ(String(tagID.UUID.String()))),
	).LIMIT(1)

	var imageTag model.ImageTags
	if err := stmt.QueryContext(ctx, s.db, &imageTag); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrImageTagNotFound
		}
		return nil, fmt.Errorf("query image tag: %w", err)
	}
	return &imageTag, nil
}

// createImageTag creates a new image-tag association.
func (s *Service) createImageTag(ctx context.Context, imageID, tagID dbtypes.UUID) (*model.ImageTags, error) {
	now := dbtypes.NewUnixMilliTimeNow()
	id := dbtypes.MustNewUUIDV7()

	row := model.ImageTags{
		ID:        &id,
		ImageID:   imageID,
		TagID:     tagID,
		CreatedAt: now,
	}

	stmt := ImageTags.INSERT(
		ImageTags.ID, ImageTags.ImageID, ImageTags.TagID, ImageTags.CreatedAt,
	).MODEL(row)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("create image tag: %w", err)
	}

	return &row, nil
}
