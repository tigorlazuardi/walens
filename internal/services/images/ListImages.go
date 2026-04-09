package images

import (
	"context"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/services/tags"
)

// ListImagesRequest describes filters for listing images.
type ListImagesRequest struct {
	Adult            *bool                            `json:"adult" doc:"Filter by adult flag"`
	Favorite         *bool                            `json:"favorite" doc:"Filter by favorite flag"`
	MinWidth         *int64                           `json:"min_width" doc:"Minimum image width in pixels"`
	MaxWidth         *int64                           `json:"max_width" doc:"Maximum image width in pixels"`
	MinHeight        *int64                           `json:"min_height" doc:"Minimum image height in pixels"`
	MaxHeight        *int64                           `json:"max_height" doc:"Maximum image height in pixels"`
	MinFileSizeBytes *int64                           `json:"min_file_size_bytes" doc:"Minimum file size in bytes"`
	MaxFileSizeBytes *int64                           `json:"max_file_size_bytes" doc:"Maximum file size in bytes"`
	Search           *string                          `json:"search" doc:"Search uploader, artist, origin URL, source item identifier, and tags"`
	SourceIDs        []dbtypes.UUID                   `json:"source_ids" doc:"Filter by source IDs"`
	Pagination       *dbtypes.CursorPaginationRequest `json:"pagination"`
}

// ListImagesResponse returns the paginated list of images.
type ListImagesResponse struct {
	Items      []model.Images                    `json:"items" doc:"List of images"`
	Pagination *dbtypes.CursorPaginationResponse `json:"pagination"`
	Total      int64                             `json:"total" doc:"Total count of images matching filters, independent of pagination"`
}

// ListImages returns all images matching the provided filters.
func (s *Service) ListImages(ctx context.Context, req ListImagesRequest) (ListImagesResponse, error) {
	var items []model.Images
	baseCond := Bool(true)

	// Adult filter
	if req.Adult != nil {
		adultVal := int64(0)
		if *req.Adult {
			adultVal = int64(1)
		}
		baseCond = baseCond.AND(Images.IsAdult.EQ(Int64(adultVal)))
	}

	// Favorite filter
	if req.Favorite != nil {
		favVal := int64(0)
		if *req.Favorite {
			favVal = int64(1)
		}
		baseCond = baseCond.AND(Images.IsFavorite.EQ(Int64(favVal)))
	}

	// Width range filter
	if req.MinWidth != nil {
		baseCond = baseCond.AND(Images.Width.GT_EQ(Int64(*req.MinWidth)))
	}
	if req.MaxWidth != nil && *req.MaxWidth > 0 {
		baseCond = baseCond.AND(Images.Width.LT_EQ(Int64(*req.MaxWidth)))
	}

	// Height range filter
	if req.MinHeight != nil {
		baseCond = baseCond.AND(Images.Height.GT_EQ(Int64(*req.MinHeight)))
	}
	if req.MaxHeight != nil && *req.MaxHeight > 0 {
		baseCond = baseCond.AND(Images.Height.LT_EQ(Int64(*req.MaxHeight)))
	}

	// File size range filter
	if req.MinFileSizeBytes != nil {
		baseCond = baseCond.AND(Images.FileSizeBytes.GT_EQ(Int64(*req.MinFileSizeBytes)))
	}
	if req.MaxFileSizeBytes != nil && *req.MaxFileSizeBytes > 0 {
		baseCond = baseCond.AND(Images.FileSizeBytes.LT_EQ(Int64(*req.MaxFileSizeBytes)))
	}

	// Source IDs filter
	if len(req.SourceIDs) > 0 {
		sourceIDCond := Images.SourceID.EQ(String(req.SourceIDs[0].UUID.String()))
		for i := 1; i < len(req.SourceIDs); i++ {
			sourceIDCond = sourceIDCond.OR(Images.SourceID.EQ(String(req.SourceIDs[i].UUID.String())))
		}
		baseCond = baseCond.AND(sourceIDCond)
	}

	// Search filter - matches uploader, artist, origin URL, source item identifier, or tags
	if req.Search != nil {
		searchTerm := strings.TrimSpace(*req.Search)
		if searchTerm != "" {
			// Build LIKE pattern for text fields
			pattern := String("%" + searchTerm + "%")

			// Build tag LIKE pattern using normalized name
			normalizedSearch := tags.NormalizeTag(searchTerm)
			tagPattern := String("%" + normalizedSearch + "%")

			// Tag subquery with LIKE on normalized_name
			tagLikeSubquery := SELECT(ImageTags.ImageID).
				FROM(ImageTags.INNER_JOIN(Tags, ImageTags.TagID.EQ(Tags.ID))).
				WHERE(Tags.NormalizedName.LIKE(tagPattern))

			// OR across all search fields
			searchCond := Images.Uploader.LIKE(pattern).
				OR(Images.Artist.LIKE(pattern)).
				OR(Images.OriginURL.LIKE(pattern)).
				OR(Images.SourceItemIdentifier.LIKE(pattern)).
				OR(Images.ID.IN(tagLikeSubquery))

			baseCond = baseCond.AND(searchCond)
		}
	}

	// Get total count before pagination filters
	total, err := s.countImages(ctx, baseCond)
	if err != nil {
		return ListImagesResponse{}, err
	}

	// Pagination - build condition with cursor filters
	cond := baseCond
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
		return ListImagesResponse{Items: []model.Images{}, Total: total}, nil
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

	return ListImagesResponse{Items: items, Pagination: cursor, Total: total}, nil
}
