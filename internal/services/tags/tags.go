// Package tags provides tag persistence for image categorization.
package tags

import (
	"database/sql"
	"errors"
	"strings"
)

// Common errors.
var (
	ErrTagNotFound = errors.New("tag not found")
)

// Service provides tag CRUD operations.
type Service struct {
	db *sql.DB
}

// NewService creates a new tags service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// DB returns the underlying database connection.
func (s *Service) DB() *sql.DB {
	return s.db
}

// NormalizeTag normalizes a tag name for deduping.
// It trims whitespace and converts to lowercase.
func NormalizeTag(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
