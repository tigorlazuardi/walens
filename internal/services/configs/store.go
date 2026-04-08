package configs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
)

// Store atomically replaces the entire persisted config value in the database.
// This performs a whole-object replacement, not a field-by-field patch.
// Uses INSERT OR REPLACE so it works whether the row exists or not.
func (s *Service) Store(ctx context.Context, cfg *PersistedConfig) error {
	value, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config for storage: %w", err)
	}

	updatedAt := time.Now().UnixMilli()

	// Use INSERT with ON CONFLICT DO UPDATE to achieve atomic upsert
	stmt := Configs.INSERT(Configs.ID, Configs.Value, Configs.UpdatedAt).
		VALUES(int64(1), String(string(value)), Int(updatedAt)).
		ON_CONFLICT(Configs.ID).DO_UPDATE(
		SET(Configs.Value.SET(String(string(value))), Configs.UpdatedAt.SET(Int(updatedAt))),
	)

	_, err = stmt.ExecContext(ctx, s.db)
	if err != nil {
		return fmt.Errorf("replace config row: %w", err)
	}

	return nil
}
