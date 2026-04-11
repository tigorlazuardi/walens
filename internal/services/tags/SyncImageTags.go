package tags

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/walens/walens/internal/dbtypes"
)

// SyncImageTags ensures all tags exist for an image and that image-tag associations exist.
// It normalizes and deduplicates the incoming tag names, then ensures:
// 1. All tags exist (creates new ones if needed)
// 2. Image-tag join records exist for all tags
//
// The operation is additive and idempotent - it does not remove existing associations
// that are not in the incoming tags list.
func (s *Service) SyncImageTags(ctx context.Context, imageID dbtypes.UUID, tagNames []string, logger *slog.Logger) error {
	if len(tagNames) == 0 {
		return nil
	}

	// Normalize and dedupe incoming tags
	normalizedTags := normalizeAndDedupeTags(tagNames)
	if len(normalizedTags) == 0 {
		return nil
	}

	// Ensure each tag exists and create image-tag association
	for _, tagName := range normalizedTags {
		// Ensure tag exists (find or create)
		tag, err := s.EnsureTag(ctx, tagName)
		if err != nil {
			logger.Warn("failed to ensure tag", "tag", tagName, "error", err)
			continue
		}
		if tag == nil {
			// Blank tag was filtered out
			continue
		}

		// Ensure image-tag association exists
		_, err = s.EnsureImageTag(ctx, imageID, tag.ID)
		if err != nil {
			logger.Warn("failed to ensure image tag", "image_id", imageID, "tag_id", tag.ID, "error", err)
			continue
		}
	}

	return nil
}

// normalizeAndDedupeTags takes a list of tag names, normalizes them,
// filters out blanks, and returns unique tags preserving first-seen original form.
func normalizeAndDedupeTags(tagNames []string) []string {
	seen := make(map[string]string) // normalized -> original
	for _, name := range tagNames {
		normalized := NormalizeTag(name)
		if normalized == "" {
			continue // Skip blank tags
		}
		if _, exists := seen[normalized]; !exists {
			// First occurrence: preserve the original tag name
			seen[normalized] = name
		}
	}

	if len(seen) == 0 {
		return nil
	}

	result := make([]string, 0, len(seen))
	for _, original := range seen {
		result = append(result, original)
	}
	return result
}

// SyncImageTagsWithError is like SyncImageTags but returns an error on failure.
// This variant is useful when you need to propagate errors rather than log them.
func (s *Service) SyncImageTagsWithError(ctx context.Context, imageID dbtypes.UUID, tagNames []string) error {
	if len(tagNames) == 0 {
		return nil
	}

	// Normalize and dedupe incoming tags
	normalizedTags := normalizeAndDedupeTags(tagNames)
	if len(normalizedTags) == 0 {
		return nil
	}

	// Ensure each tag exists and create image-tag association
	for _, tagName := range normalizedTags {
		// Ensure tag exists (find or create)
		tag, err := s.EnsureTag(ctx, tagName)
		if err != nil {
			return fmt.Errorf("ensure tag %q: %w", tagName, err)
		}
		if tag == nil {
			// Blank tag was filtered out
			continue
		}

		// Ensure image-tag association exists
		_, err = s.EnsureImageTag(ctx, imageID, tag.ID)
		if err != nil {
			return fmt.Errorf("ensure image tag for tag %q: %w", tagName, err)
		}
	}

	return nil
}
