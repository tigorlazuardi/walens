package devices

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/walens/walens/internal/dbtypes"
)

// ErrDBUnavailable is returned when the database is not available.
var ErrDBUnavailable = errors.New("database unavailable")

// ErrDeviceNotFound is returned when the requested device does not exist.
var ErrDeviceNotFound = errors.New("device not found")

// ErrDuplicateDeviceSlug is returned when a device with the same slug already exists.
var ErrDuplicateDeviceSlug = errors.New("device with this slug already exists")

// ErrInvalidSlug is returned when the slug format is invalid.
var ErrInvalidSlug = errors.New("invalid slug: must contain only lowercase letters, numbers, and hyphens")

// ErrInvalidScreenDimensions is returned when screen dimensions are invalid.
var ErrInvalidScreenDimensions = errors.New("screen width and height must be positive")

// ErrInvalidImageBounds is returned when min bounds exceed max bounds.
var ErrInvalidImageBounds = errors.New("min image dimensions cannot exceed max dimensions")

// ErrInvalidFilesizeBounds is returned when min filesize exceeds max filesize.
var ErrInvalidFilesizeBounds = errors.New("min filesize cannot exceed max filesize")

// ErrInvalidAspectRatioTolerance is returned when aspect ratio tolerance is invalid.
var ErrInvalidAspectRatioTolerance = errors.New("aspect ratio tolerance must be between 0 and 1")

// slugRegex validates slug format: lowercase letters, numbers, and hyphens only.
var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

// DeviceRow represents a device row in the database.
type DeviceRow struct {
	ID                   dbtypes.UUID          `json:"id" doc:"Unique device identifier (UUIDv7)."`
	Name                 string                `json:"name" doc:"Human-readable device name."`
	Slug                 string                `json:"slug" doc:"URL-safe device identifier for paths."`
	ScreenWidth          int64                 `json:"screen_width" doc:"Device screen width in pixels."`
	ScreenHeight         int64                 `json:"screen_height" doc:"Device screen height in pixels."`
	MinImageWidth        int64                 `json:"min_image_width" doc:"Minimum image width filter in pixels (0 = no limit)."`
	MaxImageWidth        int64                 `json:"max_image_width" doc:"Maximum image width filter in pixels (0 = no limit)."`
	MinImageHeight       int64                 `json:"min_image_height" doc:"Minimum image height filter in pixels (0 = no limit)."`
	MaxImageHeight       int64                 `json:"max_image_height" doc:"Maximum image height filter in pixels (0 = no limit)."`
	MinFilesize          int64                 `json:"min_filesize" doc:"Minimum file size filter in bytes (0 = no limit)."`
	MaxFilesize          int64                 `json:"max_filesize" doc:"Maximum file size filter in bytes (0 = no limit)."`
	IsAdultAllowed       dbtypes.BoolInt       `json:"is_adult_allowed" doc:"Whether adult content is allowed for this device."`
	IsEnabled            dbtypes.BoolInt       `json:"is_enabled" doc:"Whether the device is active and receiving wallpapers."`
	AspectRatioTolerance float64               `json:"aspect_ratio_tolerance" doc:"Absolute aspect ratio tolerance for matching wallpapers."`
	CreatedAt            dbtypes.UnixMilliTime `json:"created_at" doc:"Device creation timestamp."`
	UpdatedAt            dbtypes.UnixMilliTime `json:"updated_at" doc:"Last modification timestamp."`
}

// CreateDeviceInput contains the fields needed to create a new device.
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

// UpdateDeviceInput contains the fields needed to update an existing device.
// All fields are required for full-object update semantics.
type UpdateDeviceInput struct {
	ID                   dbtypes.UUID `json:"id" doc:"Unique device identifier."`
	Name                 string       `json:"name" doc:"Human-readable device name."`
	Slug                 string       `json:"slug" doc:"URL-safe device identifier for paths."`
	ScreenWidth          int64        `json:"screen_width" doc:"Device screen width in pixels."`
	ScreenHeight         int64        `json:"screen_height" doc:"Device screen height in pixels."`
	MinImageWidth        int64        `json:"min_image_width" doc:"Minimum image width filter in pixels (0 = no limit)."`
	MaxImageWidth        int64        `json:"max_image_width" doc:"Maximum image width filter in pixels (0 = no limit)."`
	MinImageHeight       int64        `json:"min_image_height" doc:"Minimum image height filter in pixels (0 = no limit)."`
	MaxImageHeight       int64        `json:"max_image_height" doc:"Maximum image height filter in pixels (0 = no limit)."`
	MinFilesize          int64        `json:"min_filesize" doc:"Minimum file size filter in bytes (0 = no limit)."`
	MaxFilesize          int64        `json:"max_filesize" doc:"Maximum file size filter in bytes (0 = no limit)."`
	IsAdultAllowed       bool         `json:"is_adult_allowed" doc:"Whether adult content is allowed for this device."`
	IsEnabled            bool         `json:"is_enabled" doc:"Whether the device is active and receiving wallpapers."`
	AspectRatioTolerance float64      `json:"aspect_ratio_tolerance" doc:"Absolute aspect ratio tolerance for matching wallpapers (0-1)."`
}

// Service provides CRUD operations for devices.
type Service struct {
	db *sql.DB
}

// NewService creates a new devices service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// normalizeSlug converts a slug to lowercase and trims whitespace.
func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

// validateSlug checks if the slug format is valid.
func validateSlug(slug string) error {
	if slug == "" {
		return ErrInvalidSlug
	}
	if !slugRegex.MatchString(slug) {
		return ErrInvalidSlug
	}
	return nil
}

// validateDeviceInput validates common device input constraints.
func validateDeviceInput(input *CreateDeviceInput) error {
	// Validate slug format
	normalizedSlug := normalizeSlug(input.Slug)
	if err := validateSlug(normalizedSlug); err != nil {
		return err
	}

	// Validate screen dimensions
	if input.ScreenWidth <= 0 || input.ScreenHeight <= 0 {
		return ErrInvalidScreenDimensions
	}

	// Validate image bounds (if max is set, min must not exceed it)
	if input.MaxImageWidth > 0 && input.MinImageWidth > input.MaxImageWidth {
		return ErrInvalidImageBounds
	}
	if input.MaxImageHeight > 0 && input.MinImageHeight > input.MaxImageHeight {
		return ErrInvalidImageBounds
	}

	// Validate filesize bounds
	if input.MaxFilesize > 0 && input.MinFilesize > input.MaxFilesize {
		return ErrInvalidFilesizeBounds
	}

	// Validate aspect ratio tolerance
	if input.AspectRatioTolerance < 0 || input.AspectRatioTolerance > 1 {
		return ErrInvalidAspectRatioTolerance
	}

	return nil
}

// ListDevices returns all device rows.
func (s *Service) ListDevices(ctx context.Context) ([]DeviceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT id, name, slug, screen_width, screen_height, 
		       min_image_width, max_image_width, min_image_height, max_image_height,
		       min_filesize, max_filesize, is_adult_allowed, is_enabled, 
		       aspect_ratio_tolerance, created_at, updated_at
		FROM devices
		ORDER BY name ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()

	results := make([]DeviceRow, 0)
	for rows.Next() {
		var dev DeviceRow
		if err := rows.Scan(
			&dev.ID, &dev.Name, &dev.Slug, &dev.ScreenWidth, &dev.ScreenHeight,
			&dev.MinImageWidth, &dev.MaxImageWidth, &dev.MinImageHeight, &dev.MaxImageHeight,
			&dev.MinFilesize, &dev.MaxFilesize, &dev.IsAdultAllowed, &dev.IsEnabled,
			&dev.AspectRatioTolerance, &dev.CreatedAt, &dev.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan device row: %w", err)
		}
		results = append(results, dev)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// GetDevice returns a single device by ID.
func (s *Service) GetDevice(ctx context.Context, id dbtypes.UUID) (*DeviceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT id, name, slug, screen_width, screen_height, 
		       min_image_width, max_image_width, min_image_height, max_image_height,
		       min_filesize, max_filesize, is_adult_allowed, is_enabled, 
		       aspect_ratio_tolerance, created_at, updated_at
		FROM devices
		WHERE id = ?
	`

	var dev DeviceRow
	err := s.db.QueryRowContext(ctx, query, id.UUID.String()).Scan(
		&dev.ID, &dev.Name, &dev.Slug, &dev.ScreenWidth, &dev.ScreenHeight,
		&dev.MinImageWidth, &dev.MaxImageWidth, &dev.MinImageHeight, &dev.MaxImageHeight,
		&dev.MinFilesize, &dev.MaxFilesize, &dev.IsAdultAllowed, &dev.IsEnabled,
		&dev.AspectRatioTolerance, &dev.CreatedAt, &dev.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query device: %w", err)
	}

	return &dev, nil
}

// GetDeviceBySlug returns a single device by slug.
func (s *Service) GetDeviceBySlug(ctx context.Context, slug string) (*DeviceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT id, name, slug, screen_width, screen_height, 
		       min_image_width, max_image_width, min_image_height, max_image_height,
		       min_filesize, max_filesize, is_adult_allowed, is_enabled, 
		       aspect_ratio_tolerance, created_at, updated_at
		FROM devices
		WHERE slug = ?
	`

	var dev DeviceRow
	err := s.db.QueryRowContext(ctx, query, normalizeSlug(slug)).Scan(
		&dev.ID, &dev.Name, &dev.Slug, &dev.ScreenWidth, &dev.ScreenHeight,
		&dev.MinImageWidth, &dev.MaxImageWidth, &dev.MinImageHeight, &dev.MaxImageHeight,
		&dev.MinFilesize, &dev.MaxFilesize, &dev.IsAdultAllowed, &dev.IsEnabled,
		&dev.AspectRatioTolerance, &dev.CreatedAt, &dev.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query device by slug: %w", err)
	}

	return &dev, nil
}

// CreateDevice creates a new device row.
func (s *Service) CreateDevice(ctx context.Context, input *CreateDeviceInput) (*DeviceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	// Normalize and validate input
	input.Slug = normalizeSlug(input.Slug)
	if err := validateDeviceInput(input); err != nil {
		return nil, err
	}

	// Check for duplicate slug
	var existingID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM devices WHERE slug = ?`, input.Slug).Scan(&existingID)
	if err == nil {
		return nil, ErrDuplicateDeviceSlug
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("check duplicate slug: %w", err)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return nil, fmt.Errorf("generate UUIDv7: %w", err)
	}

	isAdultAllowed := dbtypes.BoolInt(input.IsAdultAllowed)
	isEnabled := dbtypes.BoolInt(input.IsEnabled)

	query := `
		INSERT INTO devices (
			id, name, slug, screen_width, screen_height, 
			min_image_width, max_image_width, min_image_height, max_image_height,
			min_filesize, max_filesize, is_adult_allowed, is_enabled, 
			aspect_ratio_tolerance, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		id.UUID.String(), input.Name, input.Slug, input.ScreenWidth, input.ScreenHeight,
		input.MinImageWidth, input.MaxImageWidth, input.MinImageHeight, input.MaxImageHeight,
		input.MinFilesize, input.MaxFilesize, isAdultAllowed, isEnabled,
		input.AspectRatioTolerance, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	return &DeviceRow{
		ID:                   id,
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
		IsAdultAllowed:       isAdultAllowed,
		IsEnabled:            isEnabled,
		AspectRatioTolerance: input.AspectRatioTolerance,
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

// UpdateDevice updates an existing device with full-object update semantics.
func (s *Service) UpdateDevice(ctx context.Context, input *UpdateDeviceInput) (*DeviceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	// Normalize and validate input
	input.Slug = normalizeSlug(input.Slug)

	// Validate using the same logic as create
	createInput := &CreateDeviceInput{
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
		IsAdultAllowed:       input.IsAdultAllowed,
		IsEnabled:            input.IsEnabled,
		AspectRatioTolerance: input.AspectRatioTolerance,
	}
	if err := validateDeviceInput(createInput); err != nil {
		return nil, err
	}

	// Check device exists
	var existingID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM devices WHERE id = ?`, input.ID.UUID.String()).Scan(&existingID)
	if err == sql.ErrNoRows {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check device exists: %w", err)
	}

	// Check for duplicate slug (excluding current device)
	var otherID string
	err = s.db.QueryRowContext(ctx, `SELECT id FROM devices WHERE slug = ? AND id != ?`, input.Slug, input.ID.UUID.String()).Scan(&otherID)
	if err == nil {
		return nil, ErrDuplicateDeviceSlug
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("check duplicate slug: %w", err)
	}

	now := dbtypes.NewUnixMilliTimeNow()

	isAdultAllowed := dbtypes.BoolInt(input.IsAdultAllowed)
	isEnabled := dbtypes.BoolInt(input.IsEnabled)

	query := `
		UPDATE devices
		SET name = ?, slug = ?, screen_width = ?, screen_height = ?,
		    min_image_width = ?, max_image_width = ?, min_image_height = ?, max_image_height = ?,
		    min_filesize = ?, max_filesize = ?, is_adult_allowed = ?, is_enabled = ?,
		    aspect_ratio_tolerance = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, query,
		input.Name, input.Slug, input.ScreenWidth, input.ScreenHeight,
		input.MinImageWidth, input.MaxImageWidth, input.MinImageHeight, input.MaxImageHeight,
		input.MinFilesize, input.MaxFilesize, isAdultAllowed, isEnabled,
		input.AspectRatioTolerance, now, input.ID.UUID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("update device: %w", err)
	}

	// Retrieve created_at from existing row
	var createdAt dbtypes.UnixMilliTime
	err = s.db.QueryRowContext(ctx, `SELECT created_at FROM devices WHERE id = ?`, input.ID.UUID.String()).Scan(&createdAt)
	if err != nil {
		return nil, fmt.Errorf("get created_at: %w", err)
	}

	return &DeviceRow{
		ID:                   input.ID,
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
		IsAdultAllowed:       isAdultAllowed,
		IsEnabled:            isEnabled,
		AspectRatioTolerance: input.AspectRatioTolerance,
		CreatedAt:            createdAt,
		UpdatedAt:            now,
	}, nil
}

// DeleteDevice deletes a device by ID.
func (s *Service) DeleteDevice(ctx context.Context, id dbtypes.UUID) error {
	if s.db == nil {
		return ErrDBUnavailable
	}

	// Check device exists
	var existingID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM devices WHERE id = ?`, id.UUID.String()).Scan(&existingID)
	if err == sql.ErrNoRows {
		return ErrDeviceNotFound
	}
	if err != nil {
		return fmt.Errorf("check device exists: %w", err)
	}

	// Note: CASCADE will delete device_source_subscriptions and image_assignments
	query := `DELETE FROM devices WHERE id = ?`
	_, err = s.db.ExecContext(ctx, query, id.UUID.String())
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	return nil
}
