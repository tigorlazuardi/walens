package images

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/services/tags"
)

// ListDeviceImagesRequest describes filters for listing images for a specific device.
type ListDeviceImagesRequest struct {
	DeviceID             dbtypes.UUID                     `json:"device_id" doc:"Device ID to match images for"`
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
	Pagination           *dbtypes.CursorPaginationRequest `json:"pagination"`
}

// ListDeviceImagesResponse returns the paginated list of images for a device.
type ListDeviceImagesResponse struct {
	Items      []model.Images                    `json:"items" doc:"List of images"`
	Pagination *dbtypes.CursorPaginationResponse `json:"pagination"`
}

// ListDeviceImages returns images that match a specific device according to
// the device's subscription, dimension, filesize, and adult constraints.
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

	// Check device is enabled
	if !bool(device.IsEnabled) {
		return ListDeviceImagesResponse{Items: []model.Images{}}, nil
	}

	// Build the base condition:
	// - Image must come from a source the device subscribes to (enabled subscription)
	// - Device must be enabled
	cond := Bool(true)

	// Source subscription join condition - chain AND conditions
	cond = cond.AND(DeviceSourceSubscriptions.DeviceID.EQ(String(req.DeviceID.UUID.String())))
	cond = cond.AND(DeviceSourceSubscriptions.IsEnabled.EQ(Int64(1)))
	cond = cond.AND(Images.SourceID.EQ(DeviceSourceSubscriptions.SourceID))

	// If device is_adult_allowed = false, exclude adult images
	if !bool(device.IsAdultAllowed) {
		cond = cond.AND(Images.IsAdult.EQ(Int64(0)))
	}

	// Aspect ratio tolerance check: |image_aspect - device_aspect| <= tolerance
	// We express this as: aspect_ratio >= (device_aspect - tolerance) AND aspect_ratio <= (device_aspect + tolerance)
	deviceAspectRatio := float64(device.ScreenWidth) / float64(device.ScreenHeight)
	tolerance := device.AspectRatioTolerance
	if tolerance > 0 {
		minAspect := deviceAspectRatio - tolerance
		maxAspect := deviceAspectRatio + tolerance
		cond = cond.AND(Images.AspectRatio.GT_EQ(Float(minAspect)))
		cond = cond.AND(Images.AspectRatio.LT_EQ(Float(maxAspect)))
	}

	// Image dimensions must be >= device screen dimensions
	cond = cond.AND(Images.Width.GT_EQ(Int(device.ScreenWidth)))
	cond = cond.AND(Images.Height.GT_EQ(Int(device.ScreenHeight)))

	// Device min/max image dimension constraints (when non-zero)
	if device.MinImageWidth > 0 {
		cond = cond.AND(Images.Width.GT_EQ(Int(device.MinImageWidth)))
	}
	if device.MaxImageWidth > 0 {
		cond = cond.AND(Images.Width.LT_EQ(Int(device.MaxImageWidth)))
	}
	if device.MinImageHeight > 0 {
		cond = cond.AND(Images.Height.GT_EQ(Int(device.MinImageHeight)))
	}
	if device.MaxImageHeight > 0 {
		cond = cond.AND(Images.Height.LT_EQ(Int(device.MaxImageHeight)))
	}

	// Device min/max filesize constraints (when non-zero)
	if device.MinFilesize > 0 {
		cond = cond.AND(Images.FileSizeBytes.GT_EQ(Int(device.MinFilesize)))
	}
	if device.MaxFilesize > 0 {
		cond = cond.AND(Images.FileSizeBytes.LT_EQ(Int(device.MaxFilesize)))
	}

	// Apply optional request-level filters on top

	// Adult filter (only if device allows adult content)
	if req.Adult != nil && bool(device.IsAdultAllowed) {
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

	// Width range filter (request-level overrides device-level if stricter)
	if req.MinWidth != nil {
		// Take the max of device constraint and request constraint
		minW := *req.MinWidth
		if device.MinImageWidth > 0 && device.MinImageWidth > minW {
			minW = device.MinImageWidth
		}
		cond = cond.AND(Images.Width.GT_EQ(Int(minW)))
	}
	if req.MaxWidth != nil && *req.MaxWidth > 0 {
		// Take the min of device constraint and request constraint
		maxW := *req.MaxWidth
		if device.MaxImageWidth > 0 && device.MaxImageWidth < maxW {
			maxW = device.MaxImageWidth
		}
		cond = cond.AND(Images.Width.LT_EQ(Int(maxW)))
	}

	// Height range filter
	if req.MinHeight != nil {
		minH := *req.MinHeight
		if device.MinImageHeight > 0 && device.MinImageHeight > minH {
			minH = device.MinImageHeight
		}
		cond = cond.AND(Images.Height.GT_EQ(Int(minH)))
	}
	if req.MaxHeight != nil && *req.MaxHeight > 0 {
		maxH := *req.MaxHeight
		if device.MaxImageHeight > 0 && device.MaxImageHeight < maxH {
			maxH = device.MaxImageHeight
		}
		cond = cond.AND(Images.Height.LT_EQ(Int(maxH)))
	}

	// File size range filter
	if req.MinFileSizeBytes != nil {
		minF := *req.MinFileSizeBytes
		if device.MinFilesize > 0 && device.MinFilesize > minF {
			minF = device.MinFilesize
		}
		cond = cond.AND(Images.FileSizeBytes.GT_EQ(Int(minF)))
	}
	if req.MaxFileSizeBytes != nil && *req.MaxFileSizeBytes > 0 {
		maxF := *req.MaxFileSizeBytes
		if device.MaxFilesize > 0 && device.MaxFilesize < maxF {
			maxF = device.MaxFilesize
		}
		cond = cond.AND(Images.FileSizeBytes.LT_EQ(Int(maxF)))
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

	// Query with device_source_subscriptions join
	var items []model.Images
	stmt := SELECT(Images.AllColumns).
		FROM(Images.INNER_JOIN(DeviceSourceSubscriptions, Images.SourceID.EQ(DeviceSourceSubscriptions.SourceID))).
		WHERE(cond).
		ORDER_BY(orderBy...).
		LIMIT(limit + 1).
		OFFSET(req.Pagination.GetOffset())

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		return ListDeviceImagesResponse{}, huma.Error500InternalServerError("failed to list device images", err)
	}
	if len(items) == 0 {
		return ListDeviceImagesResponse{Items: []model.Images{}}, nil
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

	return ListDeviceImagesResponse{Items: items, Pagination: cursor}, nil
}
