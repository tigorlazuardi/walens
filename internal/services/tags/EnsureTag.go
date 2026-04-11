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

// EnsureTag finds an existing tag by normalized name, or creates a new one if not found.
// The returned tag has Name set to the first non-blank original name seen for that
// normalized name, and NormalizedName set to the canonical lowercase-trimmed form.
func (s *Service) EnsureTag(ctx context.Context, name string) (*model.Tags, error) {
	normalized := NormalizeTag(name)
	if normalized == "" {
		// Ignore blank tags after normalization
		return nil, nil
	}

	// Try to find existing tag by normalized name
	existing, err := s.getTagByNormalizedName(ctx, normalized)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrTagNotFound) {
		return nil, fmt.Errorf("check existing tag: %w", err)
	}

	// Create new tag
	return s.createTag(ctx, name, normalized)
}

// getTagByNormalizedName looks up a tag by its normalized name.
func (s *Service) getTagByNormalizedName(ctx context.Context, normalizedName string) (*model.Tags, error) {
	stmt := SELECT(
		Tags.ID, Tags.Name, Tags.NormalizedName, Tags.CreatedAt, Tags.UpdatedAt,
	).FROM(
		Tags,
	).WHERE(
		Tags.NormalizedName.EQ(String(normalizedName)),
	).LIMIT(1)

	var tag model.Tags
	if err := stmt.QueryContext(ctx, s.db, &tag); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("query tag: %w", err)
	}
	return &tag, nil
}

// createTag creates a new tag with the given original name and normalized form.
func (s *Service) createTag(ctx context.Context, name, normalizedName string) (*model.Tags, error) {
	now := dbtypes.NewUnixMilliTimeNow()
	id := dbtypes.MustNewUUIDV7()

	// Use the original name as provided (first non-blank), but normalized for storage
	row := model.Tags{
		ID:             id,
		Name:           name,
		NormalizedName: normalizedName,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	stmt := Tags.INSERT(
		Tags.ID, Tags.Name, Tags.NormalizedName, Tags.CreatedAt, Tags.UpdatedAt,
	).MODEL(row)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}

	return &row, nil
}
