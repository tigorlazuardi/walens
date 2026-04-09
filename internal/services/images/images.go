package images

import (
	"database/sql"
	"errors"
)

var (
	ErrImageNotFound       = errors.New("image not found")
	ErrAssignmentNotFound  = errors.New("image assignment not found")
	ErrLocationNotFound    = errors.New("image location not found")
	ErrThumbnailNotFound   = errors.New("image thumbnail not found")
	ErrDeviceNotFound      = errors.New("device not found")
	ErrNoSubscribedDevices = errors.New("no subscribed devices found")
)

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// DB returns the underlying database connection.
func (s *Service) DB() *sql.DB {
	return s.db
}
