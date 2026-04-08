// Package dbtypes provides custom database wrapper types used by Go-Jet
// generated models. These wrappers implement database/sql and
// driver interfaces so generated models remain compatible with
// standard sql.DB scan/write operations.
//
// Custom types also implement Huma's SchemaProvider interface for
// accurate OpenAPI schema generation.
package dbtypes

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

// =============================================================================
// BoolInt - wraps bool for INTEGER storage (is_* columns)
// =============================================================================

// BoolInt wraps bool for storage as INTEGER (0/1) in SQLite.
// Use for columns prefixed with is_ (e.g., is_adult_allowed, is_enabled).
type BoolInt bool

// Scan implements sql.Scanner. Reads INTEGER from DB and converts to bool.
func (b *BoolInt) Scan(src interface{}) error {
	if src == nil {
		*b = false
		return nil
	}
	var val int64
	switch v := src.(type) {
	case int64:
		val = v
	case int32:
		val = int64(v)
	case int:
		val = int64(v)
	case []byte:
		if len(v) == 0 {
			val = 0
		} else {
			fmt.Sscanf(string(v), "%d", &val)
		}
	default:
		return fmt.Errorf("cannot scan %T into BoolInt", src)
	}
	*b = val != 0
	return nil
}

// Value implements driver.Valuer. Converts bool to INTEGER (0/1).
func (b BoolInt) Value() (driver.Value, error) {
	if b {
		return int64(1), nil
	}
	return int64(0), nil
}

// Schema implements huma.SchemaProvider for OpenAPI generation.
func (BoolInt) Schema(_ huma.Registry) *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeBoolean,
		Description: "Boolean stored as integer (0/1) in the database.",
	}
}

// =============================================================================
// RawJSON - wraps json.RawMessage for JSON text storage (json_* columns)
// =============================================================================

// RawJSON wraps json.RawMessage for storage as TEXT in SQLite.
// Use for columns prefixed with json_ (e.g., json_meta, json_input, json_result).
type RawJSON json.RawMessage

// Scan implements sql.Scanner. Reads TEXT from DB and stores as raw JSON.
func (r *RawJSON) Scan(src interface{}) error {
	if src == nil {
		*r = RawJSON("null")
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*r = RawJSON(v)
	case string:
		*r = RawJSON(v)
	default:
		return fmt.Errorf("cannot scan %T into RawJSON", src)
	}
	return nil
}

// Value implements driver.Valuer. Returns JSON bytes for storage.
func (r RawJSON) Value() (driver.Value, error) {
	if r == nil {
		return []byte("null"), nil
	}
	return []byte(r), nil
}

// Schema implements huma.SchemaProvider for OpenAPI generation.
func (RawJSON) Schema(_ huma.Registry) *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeString,
		Format:      "json",
		Description: "Raw JSON object/array stored as text in the database.",
	}
}

// =============================================================================
// UnixMilliTime - wraps time.Time backed by INTEGER Unix milliseconds
// =============================================================================

// UnixMilliTime wraps time.Time for storage as INTEGER Unix milliseconds.
// Use for timestamp columns like created_at, updated_at, run_after,
// started_at, finished_at.
type UnixMilliTime struct {
	time.Time
}

// NewUnixMilliTime creates a UnixMilliTime from a time.Time.
func NewUnixMilliTime(t time.Time) UnixMilliTime {
	return UnixMilliTime{Time: t}
}

// NewUnixMilliTimeNow creates a UnixMilliTime for the current time.
func NewUnixMilliTimeNow() UnixMilliTime {
	return NewUnixMilliTime(time.Now().UTC())
}

// Scan implements sql.Scanner. Reads INTEGER milliseconds from DB.
func (t *UnixMilliTime) Scan(src interface{}) error {
	if src == nil {
		t.Time = time.Time{}
		return nil
	}
	var ms int64
	switch v := src.(type) {
	case int64:
		ms = v
	case int32:
		ms = int64(v)
	case int:
		ms = int64(v)
	case []byte:
		fmt.Sscanf(string(v), "%d", &ms)
	default:
		return fmt.Errorf("cannot scan %T into UnixMilliTime", src)
	}
	if ms == 0 {
		t.Time = time.Time{}
		return nil
	}
	t.Time = time.UnixMilli(ms).UTC()
	return nil
}

// Value implements driver.Valuer. Returns Unix milliseconds as int64.
func (t UnixMilliTime) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return int64(0), nil
	}
	return t.Time.UnixMilli(), nil
}

// Schema implements huma.SchemaProvider for OpenAPI generation.
func (UnixMilliTime) Schema(_ huma.Registry) *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeString,
		Format:      "date-time",
		Description: "Timestamp stored as Unix milliseconds in the database. Serialized as RFC3339 in JSON.",
	}
}

// =============================================================================
// UnixMilliDuration - wraps time.Duration backed by INTEGER milliseconds
// =============================================================================

// UnixMilliDuration wraps time.Duration for storage as INTEGER milliseconds.
// Use for columns ending with _ms (e.g., duration_ms).
type UnixMilliDuration time.Duration

// NewUnixMilliDuration creates a UnixMilliDuration from a time.Duration.
func NewUnixMilliDuration(d time.Duration) UnixMilliDuration {
	return UnixMilliDuration(d)
}

// Scan implements sql.Scanner. Reads INTEGER milliseconds from DB.
func (d *UnixMilliDuration) Scan(src interface{}) error {
	if src == nil {
		*d = 0
		return nil
	}
	var ms int64
	switch v := src.(type) {
	case int64:
		ms = v
	case int32:
		ms = int64(v)
	case int:
		ms = int64(v)
	case []byte:
		fmt.Sscanf(string(v), "%d", &ms)
	case string:
		val, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		*d = UnixMilliDuration(val)
		return nil
	default:
		return fmt.Errorf("cannot scan %T into UnixMilliDuration", src)
	}
	*d = UnixMilliDuration(time.Duration(ms) * time.Millisecond)
	return nil
}

// Value implements driver.Valuer. Returns duration milliseconds as int64.
func (d UnixMilliDuration) Value() (driver.Value, error) {
	return int64(time.Duration(d) / time.Millisecond), nil
}

// Milliseconds returns the duration as integer milliseconds.
func (d UnixMilliDuration) Milliseconds() int64 {
	return int64(time.Duration(d) / time.Millisecond)
}

// Schema implements huma.SchemaProvider for OpenAPI generation.
func (UnixMilliDuration) Schema(_ huma.Registry) *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeInteger,
		Format:      "int64",
		Description: "Duration in milliseconds. Input accepts integer milliseconds or Go duration string (e.g., '5s', '1h30m').",
	}
}

// =============================================================================
// UUID - wraps uuid.UUID for TEXT storage
// =============================================================================

// UUID wraps google/uuid.UUID for consistent storage and API use.
type UUID struct {
	uuid.UUID
}

// NewUUID creates a UUID from a google/uuid.UUID.
func NewUUID(u uuid.UUID) UUID {
	return UUID{UUID: u}
}

// NewUUIDFromString parses a string into a UUID.
func NewUUIDFromString(s string) (UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return UUID{}, err
	}
	return UUID{UUID: u}, nil
}

// NewUUIDV7 generates a new UUIDv7 (time-ordered UUID).
func NewUUIDV7() (UUID, error) {
	u, err := uuid.NewV7()
	if err != nil {
		return UUID{}, err
	}
	return UUID{UUID: u}, nil
}

// MustNewUUIDV7 generates a new UUIDv7 and panics on error.
func MustNewUUIDV7() UUID {
	u, err := uuid.NewV7()
	if err != nil {
		panic("failed to generate UUIDv7: " + err.Error())
	}
	return UUID{UUID: u}
}

// Scan implements sql.Scanner. Reads TEXT from DB and parses as UUID.
func (u *UUID) Scan(src interface{}) error {
	if src == nil {
		u.UUID = uuid.UUID{}
		return nil
	}
	var uuidStr string
	switch v := src.(type) {
	case []byte:
		uuidStr = string(v)
	case string:
		uuidStr = v
	default:
		return fmt.Errorf("cannot scan %T into UUID", src)
	}
	if uuidStr == "" {
		u.UUID = uuid.UUID{}
		return nil
	}
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return fmt.Errorf("invalid UUID %q: %w", uuidStr, err)
	}
	u.UUID = parsed
	return nil
}

// Value implements driver.Valuer. Returns UUID as string.
func (u UUID) Value() (driver.Value, error) {
	if u.UUID == uuid.Nil {
		return nil, nil
	}
	return u.UUID.String(), nil
}

// Schema implements huma.SchemaProvider for OpenAPI generation.
func (UUID) Schema(_ huma.Registry) *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeString,
		Format:      "uuid",
		Description: "UUIDv7 identifier for internal database records.",
	}
}

// =============================================================================
// Null types for nullable columns
// =============================================================================

// NullBoolInt is the nullable version of BoolInt.
type NullBoolInt struct {
	BoolInt
	Valid bool
}

// Scan implements sql.Scanner.
func (n *NullBoolInt) Scan(src interface{}) error {
	if src == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return n.BoolInt.Scan(src)
}

// NullRawJSON is the nullable version of RawJSON.
type NullRawJSON struct {
	RawJSON
	Valid bool
}

// Scan implements sql.Scanner.
func (n *NullRawJSON) Scan(src interface{}) error {
	if src == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return n.RawJSON.Scan(src)
}

// NullUnixMilliTime is the nullable version of UnixMilliTime.
type NullUnixMilliTime struct {
	UnixMilliTime
	Valid bool
}

// Scan implements sql.Scanner.
func (n *NullUnixMilliTime) Scan(src interface{}) error {
	if src == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return n.UnixMilliTime.Scan(src)
}

// NullUnixMilliDuration is the nullable version of UnixMilliDuration.
type NullUnixMilliDuration struct {
	UnixMilliDuration
	Valid bool
}

// Scan implements sql.Scanner.
func (n *NullUnixMilliDuration) Scan(src interface{}) error {
	if src == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return n.UnixMilliDuration.Scan(src)
}

// NullUUID is the nullable version of UUID.
type NullUUID struct {
	UUID
	Valid bool
}

// Scan implements sql.Scanner.
func (n *NullUUID) Scan(src interface{}) error {
	if src == nil {
		n.Valid = false
		return nil
	}
	n.Valid = true
	return n.UUID.Scan(src)
}

// =============================================================================
// JSON marshaling for transport
// =============================================================================

// MarshalJSON implements json.Marshaler for UnixMilliTime.
func (t UnixMilliTime) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time)
}

// UnmarshalJSON implements json.Unmarshaler for UnixMilliTime.
func (t *UnixMilliTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		t.Time = time.Time{}
		return nil
	}
	var inner time.Time
	if err := json.Unmarshal(data, &inner); err != nil {
		return err
	}
	t.Time = inner.UTC()
	return nil
}

// MarshalJSON for UnixMilliDuration outputs integer milliseconds.
func (d UnixMilliDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(time.Duration(d) / time.Millisecond))
}

// UnmarshalJSON accepts both integer milliseconds and duration string.
func (d *UnixMilliDuration) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		*d = 0
		return nil
	}
	// Try as integer milliseconds first
	var ms int64
	if err := json.Unmarshal(data, &ms); err == nil {
		*d = UnixMilliDuration(time.Duration(ms) * time.Millisecond)
		return nil
	}
	// Fall back to duration string parsing (e.g., "5s", "1h30m")
	var durStr string
	if err := json.Unmarshal(data, &durStr); err != nil {
		return fmt.Errorf("cannot unmarshal %s as duration: %w", string(data), err)
	}
	dur, err := time.ParseDuration(durStr)
	if err != nil {
		return fmt.Errorf("cannot parse duration %q: %w", durStr, err)
	}
	*d = UnixMilliDuration(dur)
	return nil
}

// =============================================================================
// Compile-time interface checks
// =============================================================================

var (
	_ sql.Scanner         = (*BoolInt)(nil)
	_ driver.Valuer       = BoolInt(false)
	_ huma.SchemaProvider = BoolInt(false)

	_ sql.Scanner         = (*RawJSON)(nil)
	_ driver.Valuer       = RawJSON(nil)
	_ huma.SchemaProvider = RawJSON(nil)

	_ sql.Scanner         = (*UnixMilliTime)(nil)
	_ driver.Valuer       = UnixMilliTime{}
	_ huma.SchemaProvider = UnixMilliTime{}

	_ sql.Scanner         = (*UnixMilliDuration)(nil)
	_ driver.Valuer       = UnixMilliDuration(0)
	_ huma.SchemaProvider = UnixMilliDuration(0)
	_ json.Marshaler      = UnixMilliDuration(0)
	_ json.Unmarshaler    = (*UnixMilliDuration)(nil)

	_ sql.Scanner         = (*UUID)(nil)
	_ driver.Valuer       = UUID{}
	_ huma.SchemaProvider = UUID{}
)
