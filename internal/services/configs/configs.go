package configs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/config"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
)

// ErrConfigNotFound is returned when the config row does not exist or is effectively empty.
var ErrConfigNotFound = errors.New("config not found")

// Service provides access to the persisted config store.
type Service struct {
	db *sql.DB
}

// NewService creates a new config service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// PersistedConfig represents the subset of application configuration that is persisted
// to the database. Only fields that make sense to change at runtime are persisted.
// Auth configuration and bootstrap-only fields are intentionally excluded.
type PersistedConfig struct {
	DataDir  string `json:"data_dir" doc:"Directory for storing application data."`
	LogLevel string `json:"log_level" doc:"Logging level (debug, info, warn, error)."`
}

// DefaultPersistedConfig returns the built-in persisted config defaults.
// Bootstrap config values may overlay these defaults before any DB config is loaded.
func DefaultPersistedConfig() *PersistedConfig {
	return &PersistedConfig{
		DataDir:  "./data",
		LogLevel: "info",
	}
}

// ApplyBootstrapConfig overlays bootstrap/env-derived persisted fields on top of
// the built-in defaults before any DB config is loaded.
// Note: BasePath is NOT applied from bootstrap config because it is bootstrap-only
// and must come from environment or command-line flags, not from persisted storage.
func (c *PersistedConfig) ApplyBootstrapConfig(cfg *config.Config) {
	c.DataDir = cfg.DataDir
	c.LogLevel = cfg.LogLevel
}

// Load reads the persisted config from the database.
// If the config row does not exist or contains empty/blank JSON, it returns
// ErrConfigNotFound. Use BootstrapDefault to initialize defaults.
func (s *Service) Load(ctx context.Context) (*PersistedConfig, error) {
	stmt := SELECT(Configs.Value, Configs.UpdatedAt).
		FROM(Configs).
		WHERE(Configs.ID.EQ(Int(1)))

	var cfg model.Configs
	err := stmt.QueryContext(ctx, s.db, &cfg)
	if err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrConfigNotFound
		}
		return nil, err
	}

	// Treat empty or whitespace-only JSON as missing
	trimmed := []byte(cfg.Value)
	for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t' || trimmed[0] == '\n' || trimmed[0] == '\r') {
		trimmed = trimmed[1:]
	}
	if len(trimmed) == 0 || string(trimmed) == "{}" {
		return nil, ErrConfigNotFound
	}

	var persisted PersistedConfig
	if err := json.Unmarshal([]byte(cfg.Value), &persisted); err != nil {
		return nil, err
	}

	return &persisted, nil
}
