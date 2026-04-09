package images

import (
	"context"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/services/tags"
)

// ListImagesRequest describes filters for listing images.
type ListImagesRequest struct {
	Adult                *bool                            `json:"adult" doc:"Filter by adult flag"`
	Favorite             *bool                            `json:"favorite" doc:"Filter by favorite flag"`
	MinWidth             *int64                           `json:"min_width" doc:"Minimum image width in pixels"`
	MaxWidth             *int64                           `json:"max_width" doc:"Maximum image width in pixels"`
	MinHeight            *int64                           `json:"min_height" doc:"Minimum image height in pixels"`
	MaxHeight            *int64                           `json:"max_height" doc:"Maximum image height in pixels"`
	MinFileSizeBytes     *int64                           `json:"min_file_size_bytes" doc:"Minimum file size in bytes"`
	MaxFileSizeBytes     *int64                           `json:"max_file_size_bytes" doc:"Maximum file size in bytes"`
	Uploader             *string                          `json:"uploader" doc:"Filter by uploader name (LIKE pattern)"`
	Artist               *string                          `json:"artist" doc:"Filter by artist name (LIKE pattern)"`
	OriginURL            *string                          `json:"origin_url" doc:"Filter by origin URL (LIKE pattern)"`
	SourceItemIdentifier *string                          `json:"source_item_identifier" doc:"Filter by source item identifier (LIKE pattern)"`
	TagNames             []string                         `json:"tag_names" doc:"Filter by tag names (ANY match)"`
	SourceIDs            []dbtypes.UUID                   `json:"source_ids" doc:"Filter by source IDs"`
	Pagination           *dbtypes.CursorPaginationRequest `json:"pagination"`
}

// ListImagesResponse returns the paginated list of images.
type ListImagesResponse struct {
	Items      []model.Images                    `json:"items" doc:"List of images"`
	Pagination *dbtypes.CursorPaginationResponse `json:"pagination"`
}

// ListImages returns all images matching the provided filters.
func (s *Service) ListImages(ctx context.Context, req ListImagesRequest) (ListImagesResponse, error) {
	var items []model.Images
	cond := Bool(true)

	// Adult filter
	if req.Adult != nil {
		adultVal := int64(0)
		if *req.Adult {
			adultVal = int64(1)
		}
		cond = cond.AND(Images.IsAdult.EQ(Int64(adultVal)))
	}

	// Favorite filter
	if req.Favorite != nil {
		favVal := int64(0)
		if *req.Favorite {
			favVal = int64(1)
		}
		cond = cond.AND(Images.IsFavorite.EQ(Int64(favVal)))
	}

	// Width range filter
	if req.MinWidth != nil {
		cond = cond.AND(Images.Width.GT_EQ(Int64(*req.MinWidth)))
	}
	if req.MaxWidth != nil && *req.MaxWidth > 0 {
		cond = cond.AND(Images.Width.LT_EQ(Int64(*req.MaxWidth)))
	}

	// Height range filter
	if req.MinHeight != nil {
		cond = cond.AND(Images.Height.GT_EQ(Int64(*req.MinHeight)))
	}
	if req.MaxHeight != nil && *req.MaxHeight > 0 {
		cond = cond.AND(Images.Height.LT_EQ(Int64(*req.MaxHeight)))
	}

	// File size range filter
	if req.MinFileSizeBytes != nil {
		cond = cond.AND(Images.FileSizeBytes.GT_EQ(Int64(*req.MinFileSizeBytes)))
	}
	if req.MaxFileSizeBytes != nil && *req.MaxFileSizeBytes > 0 {
		cond = cond.AND(Images.FileSizeBytes.LT_EQ(Int64(*req.MaxFileSizeBytes)))
	}

	// LIKE-style text filters
	if req.Uploader != nil && *req.Uploader != "" {
		pattern := String("%" + *req.Uploader + "%")
		cond = cond.AND(Images.Uploader.LIKE(pattern))
	}
	if req.Artist != nil && *req.Artist != "" {
		pattern := String("%" + *req.Artist + "%")
		cond = cond.AND(Images.Artist.LIKE(pattern))
	}
	if req.OriginURL != nil && *req.OriginURL != "" {
		pattern := String("%" + *req.OriginURL + "%")
		cond = cond.AND(Images.OriginURL.LIKE(pattern))
	}
	if req.SourceItemIdentifier != nil && *req.SourceItemIdentifier != "" {
		pattern := String("%" + *req.SourceItemIdentifier + "%")
		cond = cond.AND(Images.SourceItemIdentifier.LIKE(pattern))
	}

	// Source IDs filter
	if len(req.SourceIDs) > 0 {
		sourceIDCond := Images.SourceID.EQ(String(req.SourceIDs[0].UUID.String()))
		for i := 1; i < len(req.SourceIDs); i++ {
			sourceIDCond = sourceIDCond.OR(Images.SourceID.EQ(String(req.SourceIDs[i].UUID.String())))
		}
		cond = cond.AND(sourceIDCond)
	}

	// Tag filter - ANY match through image_tags join
	// Normalize and deduplicate tag names, skipping blanks
	normalizedTags := make([]string, 0, len(req.TagNames))
	seen := make(map[string]struct{}, len(req.TagNames))
	for _, tagName := range req.TagNames {
		normalized := tags.NormalizeTag(tagName)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; !ok {
			seen[normalized] = struct{}{}
			normalizedTags = append(normalizedTags, normalized)
		}
	}
	if len(normalizedTags) > 0 {
		tagCond := Tags.NormalizedName.EQ(String(normalizedTags[0]))
		for i := 1; i < len(normalizedTags); i++ {
			tagCond = tagCond.OR(Tags.NormalizedName.EQ(String(normalizedTags[i])))
		}
		tagStmt := SELECT(ImageTags.ImageID).
			FROM(ImageTags.INNER_JOIN(Tags, ImageTags.TagID.EQ(Tags.ID))).
			WHERE(tagCond)
		cond = cond.AND(Images.ID.IN(tagStmt))
	}

	// Pagination
	next := req.Pagination.NextToken()
	prev := req.Pagination.PrevToken()
	isPrev := next == "" && prev != ""
	if next != "" {
		cond = cond.AND(Images.ID.GT(String(next)))
	}
	if isPrev {
		cond = cond.AND(Images.ID.LT(String(prev)))
	}

	// Build order by clause
	orderBy, err := req.Pagination.BuildOrderByClause(Images.AllColumns)
	if err != nil {
		return ListImagesResponse{}, err
	}
	if len(orderBy) == 0 {
		// Default sort: CreatedAt DESC with ID as tie-breaker
		orderBy = append(orderBy, Images.CreatedAt.DESC())
	}
	if isPrev {
		orderBy = append(orderBy, Images.ID.DESC())
	} else {
		orderBy = append(orderBy, Images.ID.ASC())
	}

	limit := req.Pagination.GetLimitOrDefault(20, 100)
	stmt := SELECT(Images.AllColumns).
		FROM(Images).
		WHERE(cond).
		ORDER_BY(orderBy...).
		LIMIT(limit + 1).
		OFFSET(req.Pagination.GetOffset())

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		return ListImagesResponse{}, huma.Error500InternalServerError("failed to list images", err)
	}
	if len(items) == 0 {
		return ListImagesResponse{Items: []model.Images{}}, nil
	}

	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	cursor := &dbtypes.CursorPaginationResponse{}
	if isPrev {
		slices.Reverse(items)
	}
	if hasMore {
		cursor.Next = items[len(items)-1].ID
	}
	if next != "" {
		cursor.Prev = items[0].ID
	}

	return ListImagesResponse{Items: items, Pagination: cursor}, nil
}
