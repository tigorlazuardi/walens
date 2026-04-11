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

type UpdateDeviceRequest struct {
	ID                   dbtypes.UUID `json:"id" required:"true" doc:"Unique device identifier."`
	Name                 string       `json:"name" required:"true" doc:"Human-readable device name."`
	Slug                 string       `json:"slug" required:"true" doc:"URL-safe device identifier for paths."`
	ScreenWidth          int64        `json:"screen_width" required:"true" doc:"Device screen width in pixels."`
	ScreenHeight         int64        `json:"screen_height" required:"true" doc:"Device screen height in pixels."`
	MinImageWidth        int64        `json:"min_image_width" required:"true" doc:"Minimum image width filter in pixels (0 = no limit)."`
	MaxImageWidth        int64        `json:"max_image_width" required:"true" doc:"Maximum image width filter in pixels (0 = no limit)."`
	MinImageHeight       int64        `json:"min_image_height" required:"true" doc:"Minimum image height filter in pixels (0 = no limit)."`
	MaxImageHeight       int64        `json:"max_image_height" required:"true" doc:"Maximum image height filter in pixels (0 = no limit)."`
	MinFilesize          int64        `json:"min_filesize" required:"true" doc:"Minimum file size filter in bytes (0 = no limit)."`
	MaxFilesize          int64        `json:"max_filesize" required:"true" doc:"Maximum file size filter in bytes (0 = no limit)."`
	IsAdultAllowed       bool         `json:"is_adult_allowed" required:"true" doc:"Whether adult content is allowed for this device."`
	IsEnabled            bool         `json:"is_enabled" required:"true" doc:"Whether the device is active and receiving wallpapers."`
	AspectRatioTolerance float64      `json:"aspect_ratio_tolerance" required:"true" doc:"Absolute aspect ratio tolerance for matching wallpapers (0-1)."`
}

type UpdateDeviceResponse = model.Devices

func (s *Service) UpdateDevice(ctx context.Context, req UpdateDeviceRequest) (UpdateDeviceResponse, error) {
	req.Slug = normalizeSlug(req.Slug)
	createInput := &CreateDeviceRequest{
		Name: req.Name, Slug: req.Slug, ScreenWidth: req.ScreenWidth, ScreenHeight: req.ScreenHeight,
		MinImageWidth: req.MinImageWidth, MaxImageWidth: req.MaxImageWidth,
		MinImageHeight: req.MinImageHeight, MaxImageHeight: req.MaxImageHeight,
		MinFilesize: req.MinFilesize, MaxFilesize: req.MaxFilesize,
		IsAdultAllowed: req.IsAdultAllowed, IsEnabled: req.IsEnabled, AspectRatioTolerance: req.AspectRatioTolerance,
	}
	if err := validateDeviceInput(createInput); err != nil {
		if errors.Is(err, ErrInvalidSlug) {
			return UpdateDeviceResponse{}, huma.Error400BadRequest("invalid slug: must contain only lowercase letters, numbers, and hyphens", err)
		}
		if errors.Is(err, ErrInvalidScreenDimensions) {
			return UpdateDeviceResponse{}, huma.Error400BadRequest("screen width and height must be positive", err)
		}
		if errors.Is(err, ErrInvalidImageBounds) {
			return UpdateDeviceResponse{}, huma.Error400BadRequest("min image dimensions cannot exceed max dimensions", err)
		}
		if errors.Is(err, ErrInvalidFilesizeBounds) {
			return UpdateDeviceResponse{}, huma.Error400BadRequest("min filesize cannot exceed max filesize", err)
		}
		if errors.Is(err, ErrInvalidAspectRatioTolerance) {
			return UpdateDeviceResponse{}, huma.Error400BadRequest("aspect ratio tolerance must be between 0 and 1", err)
		}
		return UpdateDeviceResponse{}, huma.Error500InternalServerError("failed to validate device", err)
	}
	existing, err := s.GetDevice(ctx, GetDeviceRequest{ID: req.ID})
	if err != nil {
		return UpdateDeviceResponse{}, err
	}
	duplicateCount, err := s.countDevices(ctx, Devices.Slug.EQ(String(req.Slug)).AND(Devices.ID.NOT_EQ(String(req.ID.UUID.String()))))
	if err != nil {
		return UpdateDeviceResponse{}, huma.Error500InternalServerError("failed to check duplicate device slug", err)
	}
	if duplicateCount > 0 {
		return UpdateDeviceResponse{}, huma.Error409Conflict("device with this slug already exists", ErrDuplicateDeviceSlug)
	}
	updated := existing
	updated.Name = req.Name
	updated.Slug = req.Slug
	updated.ScreenWidth = req.ScreenWidth
	updated.ScreenHeight = req.ScreenHeight
	updated.MinImageWidth = req.MinImageWidth
	updated.MaxImageWidth = req.MaxImageWidth
	updated.MinImageHeight = req.MinImageHeight
	updated.MaxImageHeight = req.MaxImageHeight
	updated.MinFilesize = req.MinFilesize
	updated.MaxFilesize = req.MaxFilesize
	updated.IsAdultAllowed = dbtypes.BoolInt(req.IsAdultAllowed)
	updated.IsEnabled = dbtypes.BoolInt(req.IsEnabled)
	updated.AspectRatioTolerance = req.AspectRatioTolerance
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()
	stmt := Devices.UPDATE(
		Devices.Name, Devices.Slug, Devices.ScreenWidth, Devices.ScreenHeight,
		Devices.MinImageWidth, Devices.MaxImageWidth, Devices.MinImageHeight, Devices.MaxImageHeight,
		Devices.MinFilesize, Devices.MaxFilesize, Devices.IsAdultAllowed, Devices.IsEnabled,
		Devices.AspectRatioTolerance, Devices.UpdatedAt,
	).MODEL(updated).WHERE(Devices.ID.EQ(String(req.ID.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return UpdateDeviceResponse{}, huma.Error500InternalServerError("failed to update device", err)
	}
	return s.GetDevice(ctx, GetDeviceRequest{ID: req.ID})
}
