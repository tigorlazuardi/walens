package images

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/services/tags"
)

// ListDeviceImagesRequest describes filters for listing images for a specific device.
type ListDeviceImagesRequest struct {
	DeviceID         dbtypes.UUID                     `json:"device_id" doc:"Device ID to match images for"`
	Adult            *bool                            `json:"adult" doc:"Filter by adult flag"`
	Favorite         *bool                            `json:"favorite" doc:"Filter by favorite flag"`
	MinWidth         *int64                           `json:"min_width" doc:"Minimum image width in pixels"`
	MaxWidth         *int64                           `json:"max_width" doc:"Maximum image width in pixels"`
	MinHeight        *int64                           `json:"min_height" doc:"Minimum image height in pixels"`
	MaxHeight        *int64                           `json:"max_height" doc:"Maximum image height in pixels"`
	MinFileSizeBytes *int64                           `json:"min_file_size_bytes" doc:"Minimum file size in bytes"`
	MaxFileSizeBytes *int64                           `json:"max_file_size_bytes" doc:"Maximum file size in bytes"`
	Search           *string                          `json:"search" doc:"Search uploader, artist, origin URL, source item identifier, and tags"`
	Pagination       *dbtypes.CursorPaginationRequest `json:"pagination"`
}

// ListDeviceImagesResponse returns the paginated list of images for a device.
type ListDeviceImagesResponse struct {
	Items      []model.Images                    `json:"items" doc:"List of images"`
	Pagination *dbtypes.CursorPaginationResponse `json:"pagination"`
	Total      int64                             `json:"total" doc:"Total count of images matching filters, independent of pagination"`
}

// ListDeviceImages returns images that match a specific device according to
// the device's subscription, dimension, filesize, and adult constraints.
// It also includes historical images that were previously associated with
// the device (via assignments or locations), regardless of current eligibility.
func (s *Service) ListDeviceImages(ctx context.Context, req ListDeviceImagesRequest) (ListDeviceImagesResponse, error) {
	// First, load the device to get its constraints
	var device model.Devices
	deviceStmt := SELECT(Devices.AllColumns).
		FROM(Devices).
		WHERE(Devices.ID.EQ(String(req.DeviceID.UUID.String())))
	if err := deviceStmt.QueryContext(ctx, s.db, &device); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ListDeviceImagesResponse{}, huma.Error404NotFound("device not found", ErrDeviceNotFound)
		}
		return ListDeviceImagesResponse{}, huma.Error500InternalServerError("failed to load device", err)
	}

	// Build the base condition:
	// - Image must come from a source the device subscribes to (enabled subscription)
	//   OR have an existing assignment/location for this device (historical, no enabled check)
	baseCond := Bool(true)

	// Historical branch: image has assignment or location for this device
	// These don't depend on enabled flags - images should still appear even if
	// device or subscription is now disabled
	historicalAssignExists := EXISTS(
		SELECT(ImageAssignments.ImageID).
			FROM(ImageAssignments).
			WHERE(
				ImageAssignments.DeviceID.EQ(String(req.DeviceID.UUID.String())).
					AND(ImageAssignments.ImageID.EQ(Images.ID)),
			),
	)
	historicalLocationExists := EXISTS(
		SELECT(ImageLocations.ImageID).
			FROM(ImageLocations).
			WHERE(
				ImageLocations.DeviceID.EQ(String(req.DeviceID.UUID.String())).
					AND(ImageLocations.ImageID.EQ(Images.ID)),
			),
	)

	// Current eligibility branch: requires enabled device AND enabled subscription
	// Plus all the device constraint checks
	// Note: This entire branch is only valid when device.IsEnabled = true
	// Use EXISTS subquery to avoid INNER JOIN issues that would hide historical images
	currentEligibilityExists := Bool(false)
	if bool(device.IsEnabled) {
		// Build current eligibility condition as EXISTS subquery
		currentEligibilityCond := DeviceSourceSubscriptions.DeviceID.EQ(String(req.DeviceID.UUID.String())).
			AND(DeviceSourceSubscriptions.IsEnabled.EQ(Int64(1))).
			AND(Images.SourceID.EQ(DeviceSourceSubscriptions.SourceID))

		// Device adult constraint
		if !bool(device.IsAdultAllowed) {
			currentEligibilityCond = currentEligibilityCond.AND(Images.IsAdult.EQ(Int64(0)))
		}

		// Aspect ratio tolerance check: |image_aspect - device_aspect| <= tolerance
		deviceAspectRatio := float64(device.ScreenWidth) / float64(device.ScreenHeight)
		tolerance := device.AspectRatioTolerance
		if tolerance > 0 {
			minAspect := deviceAspectRatio - tolerance
			maxAspect := deviceAspectRatio + tolerance
			currentEligibilityCond = currentEligibilityCond.AND(Images.AspectRatio.GT_EQ(Float(minAspect)))
			currentEligibilityCond = currentEligibilityCond.AND(Images.AspectRatio.LT_EQ(Float(maxAspect)))
		}

		// Image dimensions must be >= device screen dimensions
		currentEligibilityCond = currentEligibilityCond.AND(Images.Width.GT_EQ(Int(device.ScreenWidth)))
		currentEligibilityCond = currentEligibilityCond.AND(Images.Height.GT_EQ(Int(device.ScreenHeight)))

		// Device min/max image dimension constraints (when non-zero)
		if device.MinImageWidth > 0 {
			currentEligibilityCond = currentEligibilityCond.AND(Images.Width.GT_EQ(Int(device.MinImageWidth)))
		}
		if device.MaxImageWidth > 0 {
			currentEligibilityCond = currentEligibilityCond.AND(Images.Width.LT_EQ(Int(device.MaxImageWidth)))
		}
		if device.MinImageHeight > 0 {
			currentEligibilityCond = currentEligibilityCond.AND(Images.Height.GT_EQ(Int(device.MinImageHeight)))
		}
		if device.MaxImageHeight > 0 {
			currentEligibilityCond = currentEligibilityCond.AND(Images.Height.LT_EQ(Int(device.MaxImageHeight)))
		}

		// Device min/max filesize constraints (when non-zero)
		if device.MinFilesize > 0 {
			currentEligibilityCond = currentEligibilityCond.AND(Images.FileSizeBytes.GT_EQ(Int(device.MinFilesize)))
		}
		if device.MaxFilesize > 0 {
			currentEligibilityCond = currentEligibilityCond.AND(Images.FileSizeBytes.LT_EQ(Int(device.MaxFilesize)))
		}

		currentEligibilityExists = EXISTS(
			SELECT(DeviceSourceSubscriptions.ID).
				FROM(DeviceSourceSubscriptions).
				WHERE(currentEligibilityCond),
		)
	}

	// Combine historical and current eligibility conditions
	baseCond = baseCond.AND(historicalAssignExists.OR(historicalLocationExists).OR(currentEligibilityExists))

	// Apply optional request-level filters on top

	// Adult filter (only if device allows adult content)
	if req.Adult != nil && bool(device.IsAdultAllowed) {
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

	// Width range filter (request-level overrides device-level if stricter)
	if req.MinWidth != nil {
		// Take the max of device constraint and request constraint
		minW := *req.MinWidth
		if device.MinImageWidth > 0 && device.MinImageWidth > minW {
			minW = device.MinImageWidth
		}
		baseCond = baseCond.AND(Images.Width.GT_EQ(Int(minW)))
	}
	if req.MaxWidth != nil && *req.MaxWidth > 0 {
		// Take the min of device constraint and request constraint
		maxW := *req.MaxWidth
		if device.MaxImageWidth > 0 && device.MaxImageWidth < maxW {
			maxW = device.MaxImageWidth
		}
		baseCond = baseCond.AND(Images.Width.LT_EQ(Int(maxW)))
	}

	// Height range filter
	if req.MinHeight != nil {
		minH := *req.MinHeight
		if device.MinImageHeight > 0 && device.MinImageHeight > minH {
			minH = device.MinImageHeight
		}
		baseCond = baseCond.AND(Images.Height.GT_EQ(Int(minH)))
	}
	if req.MaxHeight != nil && *req.MaxHeight > 0 {
		maxH := *req.MaxHeight
		if device.MaxImageHeight > 0 && device.MaxImageHeight < maxH {
			maxH = device.MaxImageHeight
		}
		baseCond = baseCond.AND(Images.Height.LT_EQ(Int(maxH)))
	}

	// File size range filter
	if req.MinFileSizeBytes != nil {
		minF := *req.MinFileSizeBytes
		if device.MinFilesize > 0 && device.MinFilesize > minF {
			minF = device.MinFilesize
		}
		baseCond = baseCond.AND(Images.FileSizeBytes.GT_EQ(Int(minF)))
	}
	if req.MaxFileSizeBytes != nil && *req.MaxFileSizeBytes > 0 {
		maxF := *req.MaxFileSizeBytes
		if device.MaxFilesize > 0 && device.MaxFilesize < maxF {
			maxF = device.MaxFilesize
		}
		baseCond = baseCond.AND(Images.FileSizeBytes.LT_EQ(Int(maxF)))
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
		return ListDeviceImagesResponse{}, err
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
		return ListDeviceImagesResponse{}, err
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

	// Query directly from Images - all eligibility checks are done via EXISTS subqueries
	var items []model.Images
	stmt := SELECT(Images.AllColumns).
		FROM(Images).
		WHERE(cond).
		ORDER_BY(orderBy...).
		LIMIT(limit + 1).
		OFFSET(req.Pagination.GetOffset())

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		return ListDeviceImagesResponse{}, huma.Error500InternalServerError("failed to list device images", err)
	}
	if len(items) == 0 {
		return ListDeviceImagesResponse{Items: []model.Images{}, Total: total}, nil
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

	return ListDeviceImagesResponse{Items: items, Pagination: cursor, Total: total}, nil
}
