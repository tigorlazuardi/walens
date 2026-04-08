package devices

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
)

var ErrDBUnavailable = errors.New("database unavailable")
var ErrDeviceNotFound = errors.New("device not found")
var ErrDuplicateDeviceSlug = errors.New("device with this slug already exists")
var ErrInvalidSlug = errors.New("invalid slug: must contain only lowercase letters, numbers, and hyphens")
var ErrInvalidScreenDimensions = errors.New("screen width and height must be positive")
var ErrInvalidImageBounds = errors.New("min image dimensions cannot exceed max dimensions")
var ErrInvalidFilesizeBounds = errors.New("min filesize cannot exceed max filesize")
var ErrInvalidAspectRatioTolerance = errors.New("aspect ratio tolerance must be between 0 and 1")

var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

type DeviceRow = model.Devices

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func normalizeSlug(slug string) string { return strings.ToLower(strings.TrimSpace(slug)) }

func validateSlug(slug string) error {
	if slug == "" || !slugRegex.MatchString(slug) {
		return ErrInvalidSlug
	}
	return nil
}

func validateDeviceInput(input *CreateDeviceInput) error {
	if err := validateSlug(normalizeSlug(input.Slug)); err != nil {
		return err
	}
	if input.ScreenWidth <= 0 || input.ScreenHeight <= 0 {
		return ErrInvalidScreenDimensions
	}
	if input.MaxImageWidth > 0 && input.MinImageWidth > input.MaxImageWidth {
		return ErrInvalidImageBounds
	}
	if input.MaxImageHeight > 0 && input.MinImageHeight > input.MaxImageHeight {
		return ErrInvalidImageBounds
	}
	if input.MaxFilesize > 0 && input.MinFilesize > input.MaxFilesize {
		return ErrInvalidFilesizeBounds
	}
	if input.AspectRatioTolerance < 0 || input.AspectRatioTolerance > 1 {
		return ErrInvalidAspectRatioTolerance
	}
	return nil
}

func (s *Service) countDevices(ctx context.Context, condition BoolExpression) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(COUNT(Devices.ID).AS("count")).FROM(Devices).WHERE(condition)
	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("count devices: %w", err)
	}
	return count.Count, nil
}
