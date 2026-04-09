package devices

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type CreateDeviceRequest struct {
	Name                 string  `json:"name" doc:"Human-readable device name."`
	Slug                 string  `json:"slug" doc:"URL-safe device identifier for paths (lowercase letters, numbers, hyphens only)."`
	ScreenWidth          int64   `json:"screen_width" doc:"Device screen width in pixels."`
	ScreenHeight         int64   `json:"screen_height" doc:"Device screen height in pixels."`
	MinImageWidth        int64   `json:"min_image_width" doc:"Minimum image width filter in pixels (0 = no limit)."`
	MaxImageWidth        int64   `json:"max_image_width" doc:"Maximum image width filter in pixels (0 = no limit)."`
	MinImageHeight       int64   `json:"min_image_height" doc:"Minimum image height filter in pixels (0 = no limit)."`
	MaxImageHeight       int64   `json:"max_image_height" doc:"Maximum image height filter in pixels (0 = no limit)."`
	MinFilesize          int64   `json:"min_filesize" doc:"Minimum file size filter in bytes (0 = no limit)."`
	MaxFilesize          int64   `json:"max_filesize" doc:"Maximum file size filter in bytes (0 = no limit)."`
	IsAdultAllowed       bool    `json:"is_adult_allowed" doc:"Whether adult content is allowed for this device."`
	IsEnabled            bool    `json:"is_enabled" doc:"Whether the device is active and receiving wallpapers."`
	AspectRatioTolerance float64 `json:"aspect_ratio_tolerance" doc:"Absolute aspect ratio tolerance for matching wallpapers (0-1)."`
}

type CreateDeviceResponse = model.Devices

func (s *Service) CreateDevice(ctx context.Context, req CreateDeviceRequest) (CreateDeviceResponse, error) {
	req.Slug = normalizeSlug(req.Slug)
	if err := validateDeviceInput(&req); err != nil {
		if errors.Is(err, ErrInvalidSlug) {
			return CreateDeviceResponse{}, huma.Error400BadRequest("invalid slug: must contain only lowercase letters, numbers, and hyphens", err)
		}
		if errors.Is(err, ErrInvalidScreenDimensions) {
			return CreateDeviceResponse{}, huma.Error400BadRequest("screen width and height must be positive", err)
		}
		if errors.Is(err, ErrInvalidImageBounds) {
			return CreateDeviceResponse{}, huma.Error400BadRequest("min image dimensions cannot exceed max dimensions", err)
		}
		if errors.Is(err, ErrInvalidFilesizeBounds) {
			return CreateDeviceResponse{}, huma.Error400BadRequest("min filesize cannot exceed max filesize", err)
		}
		if errors.Is(err, ErrInvalidAspectRatioTolerance) {
			return CreateDeviceResponse{}, huma.Error400BadRequest("aspect ratio tolerance must be between 0 and 1", err)
		}
		return CreateDeviceResponse{}, huma.Error500InternalServerError("failed to validate device", err)
	}
	duplicateCount, err := s.countDevices(ctx, Devices.Slug.EQ(String(req.Slug)))
	if err != nil {
		return CreateDeviceResponse{}, huma.Error500InternalServerError("failed to check duplicate device slug", err)
	}
	if duplicateCount > 0 {
		return CreateDeviceResponse{}, huma.Error409Conflict("device with this slug already exists", ErrDuplicateDeviceSlug)
	}
	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return CreateDeviceResponse{}, huma.Error500InternalServerError("failed to generate device id", err)
	}
	row := model.Devices{
		ID:                   &id,
		Name:                 req.Name,
		Slug:                 req.Slug,
		ScreenWidth:          req.ScreenWidth,
		ScreenHeight:         req.ScreenHeight,
		MinImageWidth:        req.MinImageWidth,
		MaxImageWidth:        req.MaxImageWidth,
		MinImageHeight:       req.MinImageHeight,
		MaxImageHeight:       req.MaxImageHeight,
		MinFilesize:          req.MinFilesize,
		MaxFilesize:          req.MaxFilesize,
		IsAdultAllowed:       dbtypes.BoolInt(req.IsAdultAllowed),
		IsEnabled:            dbtypes.BoolInt(req.IsEnabled),
		AspectRatioTolerance: req.AspectRatioTolerance,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	stmt := Devices.INSERT(
		Devices.ID, Devices.Name, Devices.Slug, Devices.ScreenWidth, Devices.ScreenHeight,
		Devices.MinImageWidth, Devices.MaxImageWidth, Devices.MinImageHeight, Devices.MaxImageHeight,
		Devices.MinFilesize, Devices.MaxFilesize, Devices.IsAdultAllowed, Devices.IsEnabled,
		Devices.AspectRatioTolerance, Devices.CreatedAt, Devices.UpdatedAt,
	).MODEL(row)
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return CreateDeviceResponse{}, huma.Error500InternalServerError("failed to create device", err)
	}
	return s.GetDevice(ctx, GetDeviceRequest{ID: id})
}
