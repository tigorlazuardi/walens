package configs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/walens/walens/internal/config"
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
	Server   PersistedServerConfig `json:"server"`
	DataDir  string                `json:"data_dir"`
	LogLevel string                `json:"log_level"`
}

// PersistedServerConfig contains server settings that are persisted.
type PersistedServerConfig struct {
	BasePath string `json:"base_path"`
}

// DefaultPersistedConfig returns the default persisted config derived from
// the bootstrap runtime config. This is used when no persisted config exists.
func DefaultPersistedConfig(cfg *config.Config) *PersistedConfig {
	return &PersistedConfig{
		Server: PersistedServerConfig{
			BasePath: cfg.Server.BasePath,
		},
		DataDir:  cfg.DataDir,
		LogLevel: cfg.LogLevel,
	}
}

// Load reads the persisted config from the database.
// If the config row does not exist or contains empty/blank JSON, it returns
// ErrConfigNotFound. Use BootstrapDefault to initialize defaults.
func (s *Service) Load(ctx context.Context) (*PersistedConfig, error) {
	var value string
	var updatedAt int64

	err := s.db.QueryRowContext(ctx,
		`SELECT value, updated_at FROM configs WHERE id = 1`,
	).Scan(&value, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query config row: %w", err)
	}

	// Treat empty or whitespace-only JSON as missing
	trimmed := []byte(value)
	for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t' || trimmed[0] == '\n' || trimmed[0] == '\r') {
		trimmed = trimmed[1:]
	}
	if len(trimmed) == 0 || string(trimmed) == "{}" {
		return nil, ErrConfigNotFound
	}

	var cfg PersistedConfig
	if err := json.Unmarshal([]byte(value), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal persisted config: %w", err)
	}

	return &cfg, nil
}

// Store atomically replaces the entire persisted config value in the database.
// This performs a whole-object replacement, not a field-by-field patch.
// Uses INSERT OR REPLACE so it works whether the row exists or not.
func (s *Service) Store(ctx context.Context, cfg *PersistedConfig) error {
	value, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config for storage: %w", err)
	}

	updatedAt := time.Now().UnixMilli()

	_, err = s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO configs (id, value, updated_at) VALUES (1, ?, ?)`,
		string(value), updatedAt,
	)
	if err != nil {
		return fmt.Errorf("replace config row: %w", err)
	}

	return nil
}

// BootstrapDefault loads the persisted config, or if absent/empty, inserts
// the provided default config and returns it. This ensures the app always
// has a valid persisted config after bootstrap.
func (s *Service) BootstrapDefault(ctx context.Context, defaultCfg *PersistedConfig) (*PersistedConfig, error) {
	existing, err := s.Load(ctx)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrConfigNotFound) {
		return nil, err
	}

	// Config row is absent or empty; inject defaults via atomic insert.
	if err := s.Store(ctx, defaultCfg); err != nil {
		return nil, fmt.Errorf("bootstrap default config: %w", err)
	}

	return defaultCfg, nil
}
