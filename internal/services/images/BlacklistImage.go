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

// BlacklistImageInput defines the input for BlacklistImage.
type BlacklistImageInput struct {
	ImageID dbtypes.UUID `json:"image_id" doc:"Image ID to blacklist"`
	Reason  *string      `json:"reason" doc:"Optional reason for blacklisting"`
}

// BlacklistImageOutput defines the output for BlacklistImage.
type BlacklistImageOutput struct {
	Blacklist *model.ImageBlacklists `json:"blacklist"`
}

// BlacklistImage adds an image to the blacklist based on its source_id and unique_identifier.
// This operation is idempotent - if the image is already blacklisted, it returns success
// with the existing blacklist entry.
func (s *Service) BlacklistImage(ctx context.Context, input BlacklistImageInput) (*BlacklistImageOutput, error) {
	// First, load the image to get source_id and unique_identifier
	var img model.Images
	getStmt := SELECT(Images.AllColumns).
		FROM(Images).
		WHERE(Images.ID.EQ(String(input.ImageID.UUID.String()))).
		LIMIT(1)
	if err := getStmt.QueryContext(ctx, s.db, &img); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("get image for blacklist: %w", err)
	}

	// Check if already blacklisted (idempotent)
	existing, err := s.getImageBlacklist(ctx, *img.SourceID, img.UniqueIdentifier)
	if err == nil && existing != nil {
		// Already blacklisted, return the existing entry
		return &BlacklistImageOutput{Blacklist: existing}, nil
	}
	if err != nil && !errors.Is(err, ErrBlacklistNotFound) {
		return nil, fmt.Errorf("check existing blacklist: %w", err)
	}

	// Create new blacklist entry
	blacklist, err := s.ensureImageBlacklist(ctx, *img.SourceID, img.UniqueIdentifier, input.Reason)
	if err != nil {
		return nil, err
	}

	return &BlacklistImageOutput{Blacklist: blacklist}, nil
}

// getImageBlacklist retrieves a blacklist entry by source_id and unique_identifier.
// Returns ErrBlacklistNotFound if no entry exists.
func (s *Service) getImageBlacklist(ctx context.Context, sourceID dbtypes.UUID, uniqueIdentifier string) (*model.ImageBlacklists, error) {
	var blacklist model.ImageBlacklists
	stmt := SELECT(ImageBlacklists.AllColumns).
		FROM(ImageBlacklists).
		WHERE(
			ImageBlacklists.SourceID.EQ(String(sourceID.UUID.String())).
				AND(ImageBlacklists.UniqueIdentifier.EQ(String(uniqueIdentifier))),
		).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &blacklist); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrBlacklistNotFound
		}
		return nil, fmt.Errorf("get image blacklist: %w", err)
	}
	return &blacklist, nil
}

// ensureImageBlacklist creates a blacklist entry if it doesn't exist.
// Returns the created or existing blacklist entry.
func (s *Service) ensureImageBlacklist(ctx context.Context, sourceID dbtypes.UUID, uniqueIdentifier string, reason *string) (*model.ImageBlacklists, error) {
	now := dbtypes.NewUnixMilliTimeNow()
	id := dbtypes.MustNewUUIDV7()

	row := model.ImageBlacklists{
		ID:               &id,
		SourceID:         sourceID,
		UniqueIdentifier: uniqueIdentifier,
		Reason:           reason,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	stmt := ImageBlacklists.INSERT(
		ImageBlacklists.ID, ImageBlacklists.SourceID, ImageBlacklists.UniqueIdentifier,
		ImageBlacklists.Reason, ImageBlacklists.CreatedAt, ImageBlacklists.UpdatedAt,
	).MODEL(row)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("ensure image blacklist: %w", err)
	}

	return &row, nil
}
