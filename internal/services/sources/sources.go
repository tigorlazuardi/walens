package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/sources"
)

// ErrDBUnavailable is returned when the database is not available.
var ErrDBUnavailable = errors.New("database unavailable")

// ErrSourceNotFound is returned when the requested source does not exist.
var ErrSourceNotFound = errors.New("source not found")

// ErrDuplicateSourceName is returned when a source with the same name already exists.
var ErrDuplicateSourceName = errors.New("source with this name already exists")

// ErrInvalidSourceType is returned when the source_type is not registered.
var ErrInvalidSourceType = errors.New("invalid source type: not registered")

// ErrRegistryUnavailable is returned when the source registry is not available.
var ErrRegistryUnavailable = errors.New("source registry unavailable")

// ErrInvalidParams is returned when params fail validation against the source implementation.
var ErrInvalidParams = errors.New("invalid params for source type")

// SourceRow represents a source row in the database.
type SourceRow struct {
	ID          dbtypes.UUID          `json:"id" doc:"Unique source identifier (UUIDv7)."`
	Name        string                `json:"name" doc:"Unique human-readable source name."`
	SourceType  string                `json:"source_type" doc:"Registered source implementation name."`
	Params      dbtypes.RawJSON       `json:"params" doc:"Source-specific configuration as JSON."`
	LookupCount int64                 `json:"lookup_count" doc:"Upstream lookup budget per run (0 = use source default)."`
	IsEnabled   dbtypes.BoolInt       `json:"is_enabled" doc:"Whether this source is active."`
	CreatedAt   dbtypes.UnixMilliTime `json:"created_at" doc:"Source creation timestamp."`
	UpdatedAt   dbtypes.UnixMilliTime `json:"updated_at" doc:"Last modification timestamp."`
}

// CreateSourceInput contains the fields needed to create a new source.
type CreateSourceInput struct {
	Name        string          `json:"name" doc:"Unique human-readable source name."`
	SourceType  string          `json:"source_type" doc:"Registered source implementation name (e.g., booru, reddit)."`
	Params      json.RawMessage `json:"params" doc:"Source-specific configuration as JSON."`
	LookupCount int64           `json:"lookup_count" doc:"Upstream lookup budget per run (0 = use source default)."`
	IsEnabled   bool            `json:"is_enabled" doc:"Whether this source is active."`
}

// UpdateSourceInput contains the fields needed to update an existing source.
// All fields are required for full-object update semantics.
type UpdateSourceInput struct {
	ID          dbtypes.UUID    `json:"id" doc:"Unique source identifier."`
	Name        string          `json:"name" doc:"Unique human-readable source name."`
	SourceType  string          `json:"source_type" doc:"Registered source implementation name."`
	Params      json.RawMessage `json:"params" doc:"Source-specific configuration as JSON."`
	LookupCount int64           `json:"lookup_count" doc:"Upstream lookup budget per run (0 = use source default)."`
	IsEnabled   bool            `json:"is_enabled" doc:"Whether this source is active."`
}

// Service provides CRUD operations for configured source rows.
type Service struct {
	db       *sql.DB
	registry *sources.Registry
}

// NewService creates a new sources service.
func NewService(db *sql.DB, registry *sources.Registry) *Service {
	return &Service{db: db, registry: registry}
}

// validateSourceType checks that source_type is registered and params are valid.
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

// ListSources returns all configured source rows.
func (s *Service) ListSources(ctx context.Context) ([]SourceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at
		FROM sources
		ORDER BY name ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query sources: %w", err)
	}
	defer rows.Close()

	results := make([]SourceRow, 0)
	for rows.Next() {
		var src SourceRow
		if err := rows.Scan(
			&src.ID, &src.Name, &src.SourceType, &src.Params,
			&src.LookupCount, &src.IsEnabled, &src.CreatedAt, &src.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source row: %w", err)
		}
		results = append(results, src)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// GetSource returns a single source by ID.
func (s *Service) GetSource(ctx context.Context, id dbtypes.UUID) (*SourceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at
		FROM sources
		WHERE id = ?
	`

	var src SourceRow
	err := s.db.QueryRowContext(ctx, query, id.UUID.String()).Scan(
		&src.ID, &src.Name, &src.SourceType, &src.Params,
		&src.LookupCount, &src.IsEnabled, &src.CreatedAt, &src.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrSourceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query source: %w", err)
	}

	return &src, nil
}

// CreateSource creates a new source row.
func (s *Service) CreateSource(ctx context.Context, input *CreateSourceInput) (*SourceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	// Validate source_type and params against registry
	if err := s.validateSourceType(input.SourceType, input.Params); err != nil {
		return nil, err
	}

	// Check for duplicate name
	var existingID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM sources WHERE name = ?`, input.Name).Scan(&existingID)
	if err == nil {
		return nil, ErrDuplicateSourceName
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("check duplicate name: %w", err)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return nil, fmt.Errorf("generate UUIDv7: %w", err)
	}

	isEnabled := dbtypes.BoolInt(input.IsEnabled)

	query := `
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		id.UUID.String(), input.Name, input.SourceType, input.Params,
		input.LookupCount, isEnabled, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert source: %w", err)
	}

	return &SourceRow{
		ID:          id,
		Name:        input.Name,
		SourceType:  input.SourceType,
		Params:      dbtypes.RawJSON(input.Params),
		LookupCount: input.LookupCount,
		IsEnabled:   isEnabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// UpdateSource updates an existing source with full-object update semantics.
func (s *Service) UpdateSource(ctx context.Context, input *UpdateSourceInput) (*SourceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	// Validate source_type and params against registry
	if err := s.validateSourceType(input.SourceType, input.Params); err != nil {
		return nil, err
	}

	// Check source exists
	var existingID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM sources WHERE id = ?`, input.ID.UUID.String()).Scan(&existingID)
	if err == sql.ErrNoRows {
		return nil, ErrSourceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check source exists: %w", err)
	}

	// Check for duplicate name (excluding current source)
	var otherID string
	err = s.db.QueryRowContext(ctx, `SELECT id FROM sources WHERE name = ? AND id != ?`, input.Name, input.ID.UUID.String()).Scan(&otherID)
	if err == nil {
		return nil, ErrDuplicateSourceName
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("check duplicate name: %w", err)
	}

	now := dbtypes.NewUnixMilliTimeNow()

	isEnabled := dbtypes.BoolInt(input.IsEnabled)

	query := `
		UPDATE sources
		SET name = ?, source_type = ?, params = ?, lookup_count = ?, is_enabled = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, query,
		input.Name, input.SourceType, input.Params,
		input.LookupCount, isEnabled, now, input.ID.UUID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("update source: %w", err)
	}

	// Retrieve created_at from existing row
	var createdAt dbtypes.UnixMilliTime
	err = s.db.QueryRowContext(ctx, `SELECT created_at FROM sources WHERE id = ?`, input.ID.UUID.String()).Scan(&createdAt)
	if err != nil {
		return nil, fmt.Errorf("get created_at: %w", err)
	}

	return &SourceRow{
		ID:          input.ID,
		Name:        input.Name,
		SourceType:  input.SourceType,
		Params:      dbtypes.RawJSON(input.Params),
		LookupCount: input.LookupCount,
		IsEnabled:   isEnabled,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}, nil
}

// DeleteSource deletes a source by ID.
func (s *Service) DeleteSource(ctx context.Context, id dbtypes.UUID) error {
	if s.db == nil {
		return ErrDBUnavailable
	}

	// Check source exists
	var existingID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM sources WHERE id = ?`, id.UUID.String()).Scan(&existingID)
	if err == sql.ErrNoRows {
		return ErrSourceNotFound
	}
	if err != nil {
		return fmt.Errorf("check source exists: %w", err)
	}

	// Note: CASCADE will delete source_schedules and device_source_subscriptions
	query := `DELETE FROM sources WHERE id = ?`
	_, err = s.db.ExecContext(ctx, query, id.UUID.String())
	if err != nil {
		return fmt.Errorf("delete source: %w", err)
	}

	return nil
}
