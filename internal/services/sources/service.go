package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	appsources "github.com/walens/walens/internal/sources"
)

var ErrDBUnavailable = errors.New("database unavailable")
var ErrSourceNotFound = errors.New("source not found")
var ErrDuplicateSourceName = errors.New("source with this name already exists")
var ErrInvalidSourceType = errors.New("invalid source type: not registered")
var ErrRegistryUnavailable = errors.New("source registry unavailable")
var ErrInvalidParams = errors.New("invalid params for source type")

type SourceRow = model.Sources

type Service struct {
	db       *sql.DB
	registry *appsources.Registry
}

func NewService(db *sql.DB, registry *appsources.Registry) *Service {
	return &Service{db: db, registry: registry}
}

func (s *Service) validateSourceType(sourceType string, params json.RawMessage) error {
	if s.registry == nil {
		return ErrRegistryUnavailable
	}

	src := s.registry.Get(sourceType)
	if src == nil {
		return ErrInvalidSourceType
	}

	if err := src.ValidateParams(params); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	return nil
}

func (s *Service) countSources(ctx context.Context, condition BoolExpression) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}

	stmt := SELECT(COUNT(Sources.ID).AS("count")).
		FROM(Sources).
		WHERE(condition)

	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("count sources: %w", err)
	}

	return count.Count, nil
}
