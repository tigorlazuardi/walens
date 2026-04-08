package devices

import (
	"context"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type CreateDeviceInput struct {
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

func (s *Service) CreateDevice(ctx context.Context, input *CreateDeviceInput) (*DeviceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}
	input.Slug = normalizeSlug(input.Slug)
	if err := validateDeviceInput(input); err != nil {
		return nil, err
	}
	duplicateCount, err := s.countDevices(ctx, Devices.Slug.EQ(String(input.Slug)))
	if err != nil {
		return nil, fmt.Errorf("check duplicate slug: %w", err)
	}
	if duplicateCount > 0 {
		return nil, ErrDuplicateDeviceSlug
	}
	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return nil, fmt.Errorf("generate UUIDv7: %w", err)
	}
	row := model.Devices{
		ID:                   &id,
		Name:                 input.Name,
		Slug:                 input.Slug,
		ScreenWidth:          input.ScreenWidth,
		ScreenHeight:         input.ScreenHeight,
		MinImageWidth:        input.MinImageWidth,
		MaxImageWidth:        input.MaxImageWidth,
		MinImageHeight:       input.MinImageHeight,
		MaxImageHeight:       input.MaxImageHeight,
		MinFilesize:          input.MinFilesize,
		MaxFilesize:          input.MaxFilesize,
		IsAdultAllowed:       dbtypes.BoolInt(input.IsAdultAllowed),
		IsEnabled:            dbtypes.BoolInt(input.IsEnabled),
		AspectRatioTolerance: input.AspectRatioTolerance,
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
		return nil, fmt.Errorf("insert device: %w", err)
	}
	return s.GetDevice(ctx, id)
}
