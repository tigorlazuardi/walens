package images

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
)

var (
	ErrImageNotFound       = errors.New("image not found")
	ErrAssignmentNotFound  = errors.New("image assignment not found")
	ErrLocationNotFound    = errors.New("image location not found")
	ErrThumbnailNotFound   = errors.New("image thumbnail not found")
	ErrDeviceNotFound      = errors.New("device not found")
	ErrNoSubscribedDevices = errors.New("no subscribed devices found")
	ErrBlacklistNotFound   = errors.New("blacklist entry not found")
)

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// DB returns the underlying database connection.
func (s *Service) DB() *sql.DB {
	return s.db
}

func (s *Service) countImages(ctx context.Context, condition BoolExpression) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(COUNT(Images.ID).AS("count")).FROM(Images).WHERE(condition)
	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("count images: %w", err)
	}
	return count.Count, nil
}
