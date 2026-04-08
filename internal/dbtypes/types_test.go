package dbtypes

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

func TestBoolInt_Scan(t *testing.T) {
	tests := []struct {
		name     string
		src      interface{}
		expected BoolInt
		wantErr  bool
	}{
		{"int64 1", int64(1), true, false},
		{"int64 0", int64(0), false, false},
		{"int 1", int(1), true, false},
		{"int 0", int(0), false, false},
		{"byte slice 1", []byte("1"), true, false},
		{"byte slice 0", []byte("0"), false, false},
		{"nil", nil, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b BoolInt
			err := b.Scan(tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if b != tt.expected {
				t.Errorf("Scan() = %v, want %v", b, tt.expected)
			}
		})
	}
}

func TestBoolInt_Value(t *testing.T) {
	tests := []struct {
		name     string
		b        BoolInt
		expected int64
	}{
		{"true", true, 1},
		{"false", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := tt.b.Value()
			if err != nil {
				t.Errorf("Value() error = %v", err)
				return
			}
			if val.(int64) != tt.expected {
				t.Errorf("Value() = %v, want %v", val, tt.expected)
			}
		})
	}
}

func TestRawJSON_Scan(t *testing.T) {
	tests := []struct {
		name     string
		src      interface{}
		expected RawJSON
		wantErr  bool
	}{
		{"byte slice", []byte(`{"key":"value"}`), RawJSON(`{"key":"value"}`), false},
		{"string", `{"key":"value"}`, RawJSON(`{"key":"value"}`), false},
		{"nil", nil, RawJSON("null"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r RawJSON
			err := r.Scan(tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(r) != string(tt.expected) {
				t.Errorf("Scan() = %v, want %v", string(r), string(tt.expected))
			}
		})
	}
}

func TestRawJSON_Value(t *testing.T) {
	r := RawJSON(`{"key":"value"}`)
	val, err := r.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
		return
	}
	if string(val.([]byte)) != `{"key":"value"}` {
		t.Errorf("Value() = %v, want %v", val, `{"key":"value"}`)
	}
}

func TestUnixMilliTime_Scan(t *testing.T) {
	// 2026-04-08T12:00:00Z in milliseconds
	ms := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC).UnixMilli()

	tests := []struct {
		name     string
		src      interface{}
		expected time.Time
		wantErr  bool
	}{
		{"int64 ms", int64(ms), time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC), false},
		{"int ms", int(ms), time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC), false},
		{"byte slice", []byte("1775649600000"), time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC), false},
		{"nil", nil, time.Time{}, false},
		{"zero", int64(0), time.Time{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ut UnixMilliTime
			err := ut.Scan(tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !ut.Time.Equal(tt.expected) {
				t.Errorf("Scan() = %v, want %v", ut.Time, tt.expected)
			}
		})
	}
}

func TestUnixMilliTime_Value(t *testing.T) {
	ts := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	ut := NewUnixMilliTime(ts)

	val, err := ut.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
		return
	}
	if val.(int64) != ts.UnixMilli() {
		t.Errorf("Value() = %v, want %v", val, ts.UnixMilli())
	}
}

func TestUnixMilliTime_ZeroValue(t *testing.T) {
	var ut UnixMilliTime
	val, err := ut.Value()
	if err != nil {
		t.Errorf("Zero Value() error = %v", err)
		return
	}
	if val.(int64) != 0 {
		t.Errorf("Zero Value() = %v, want 0", val)
	}
}

func TestUnixMilliDuration_Scan(t *testing.T) {
	tests := []struct {
		name     string
		src      interface{}
		expected time.Duration
		wantErr  bool
	}{
		{"int64 ms", int64(5000), 5 * time.Second, false},
		{"int ms", int(3000), 3 * time.Second, false},
		{"byte slice", []byte("1000"), 1 * time.Second, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d UnixMilliDuration
			err := d.Scan(tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if time.Duration(d) != tt.expected {
				t.Errorf("Scan() = %v, want %v", time.Duration(d), tt.expected)
			}
		})
	}
}

func TestUnixMilliDuration_Value(t *testing.T) {
	d := NewUnixMilliDuration(5 * time.Second)
	val, err := d.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
		return
	}
	if val.(int64) != 5000 {
		t.Errorf("Value() = %v, want 5000", val)
	}
}

func TestUnixMilliDuration_JSONMarshal(t *testing.T) {
	d := NewUnixMilliDuration(1500 * time.Millisecond)

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	if string(data) != "1500" {
		t.Fatalf("MarshalJSON() = %s, want 1500", string(data))
	}
}

func TestUnixMilliDuration_JSONUnmarshalMilliseconds(t *testing.T) {
	var d UnixMilliDuration
	if err := json.Unmarshal([]byte("2500"), &d); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if time.Duration(d) != 2500*time.Millisecond {
		t.Fatalf("UnmarshalJSON() = %v, want %v", time.Duration(d), 2500*time.Millisecond)
	}
}

func TestUnixMilliDuration_JSONUnmarshalDurationString(t *testing.T) {
	var d UnixMilliDuration
	if err := json.Unmarshal([]byte(`"1.5s"`), &d); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	if time.Duration(d) != 1500*time.Millisecond {
		t.Fatalf("UnmarshalJSON() = %v, want %v", time.Duration(d), 1500*time.Millisecond)
	}
}

func TestUnixMilliDuration_JSONUnmarshalNull(t *testing.T) {
	d := NewUnixMilliDuration(5 * time.Second)
	if err := json.Unmarshal([]byte("null"), &d); err != nil {
		t.Fatalf("UnmarshalJSON(null) error = %v", err)
	}

	if time.Duration(d) != 0 {
		t.Fatalf("UnmarshalJSON(null) = %v, want 0", time.Duration(d))
	}
}

func TestUnixMilliDuration_JSONUnmarshalInvalid(t *testing.T) {
	var d UnixMilliDuration
	if err := json.Unmarshal([]byte(`"nope"`), &d); err == nil {
		t.Fatal("UnmarshalJSON() expected error for invalid duration string")
	}
}

func TestUUID_Scan(t *testing.T) {
	validUUID := "01915f37-0187-7000-8e63-d83a9f70d5e8"
	parsed, _ := uuid.Parse(validUUID)

	tests := []struct {
		name     string
		src      interface{}
		expected uuid.UUID
		wantErr  bool
	}{
		{"valid string", validUUID, parsed, false},
		{"valid bytes", []byte(validUUID), parsed, false},
		{"nil", nil, uuid.UUID{}, false},
		{"empty string", "", uuid.UUID{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u UUID
			err := u.Scan(tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if u.UUID != tt.expected {
				t.Errorf("Scan() = %v, want %v", u.UUID, tt.expected)
			}
		})
	}
}

func TestUUID_Invalid(t *testing.T) {
	var u UUID
	err := u.Scan("not-a-uuid")
	if err == nil {
		t.Errorf("Scan() expected error for invalid UUID")
	}
}

func TestUUID_Value(t *testing.T) {
	u := NewUUID(uuid.MustParse("01915f37-0187-7000-8e63-d83a9f70d5e8"))
	val, err := u.Value()
	if err != nil {
		t.Errorf("Value() error = %v", err)
		return
	}
	if val.(string) != "01915f37-0187-7000-8e63-d83a9f70d5e8" {
		t.Errorf("Value() = %v, want UUID string", val)
	}
}

func TestUUID_NilValue(t *testing.T) {
	var u UUID
	val, err := u.Value()
	if err != nil {
		t.Errorf("Nil Value() error = %v", err)
		return
	}
	if val != nil {
		t.Errorf("Nil Value() = %v, want nil", val)
	}
}

func TestNewUUIDV7(t *testing.T) {
	u, err := NewUUIDV7()
	if err != nil {
		t.Errorf("NewUUIDV7() error = %v", err)
		return
	}
	if u.UUID.Version() != 7 {
		t.Errorf("NewUUIDV7() version = %v, want 7", u.UUID.Version())
	}
}

func TestNullTypes(t *testing.T) {
	// Test NullBoolInt
	var nb NullBoolInt
	err := nb.Scan(int64(1))
	if err != nil {
		t.Errorf("NullBoolInt Scan() error = %v", err)
	}
	if !nb.Valid {
		t.Error("NullBoolInt Valid should be true after Scan")
	}

	var nbnull NullBoolInt
	err = nbnull.Scan(nil)
	if err != nil {
		t.Errorf("NullBoolInt Scan(nil) error = %v", err)
	}
	if nbnull.Valid {
		t.Error("NullBoolInt Valid should be false after Scan(nil)")
	}

	// Test NullUUID
	var nu NullUUID
	err = nu.Scan("01915f37-0187-7000-8e63-d83a9f70d5e8")
	if err != nil {
		t.Errorf("NullUUID Scan() error = %v", err)
	}
	if !nu.Valid {
		t.Error("NullUUID Valid should be true after Scan")
	}
}

func TestUnixMilliTime_JSON(t *testing.T) {
	ts := time.Date(2026, 4, 8, 12, 0, 0, 0, time.UTC)
	ut := NewUnixMilliTime(ts)

	data, err := json.Marshal(ut)
	if err != nil {
		t.Errorf("MarshalJSON() error = %v", err)
		return
	}

	var ut2 UnixMilliTime
	err = json.Unmarshal(data, &ut2)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}
	if !ut2.Time.Equal(ut.Time) {
		t.Errorf("Round trip = %v, want %v", ut2.Time, ut.Time)
	}
}

func TestUnixMilliDuration_JSON(t *testing.T) {
	d := NewUnixMilliDuration(5 * time.Second)

	data, err := json.Marshal(d)
	if err != nil {
		t.Errorf("MarshalJSON() error = %v", err)
		return
	}
	if string(data) != "5000" {
		t.Errorf("MarshalJSON() = %v, want 5000", string(data))
	}

	var d2 UnixMilliDuration
	err = json.Unmarshal(data, &d2)
	if err != nil {
		t.Errorf("UnmarshalJSON() error = %v", err)
		return
	}
	if d2 != d {
		t.Errorf("Round trip = %v, want %v", d2, d)
	}
}

func TestUnixMilliDuration_JSON_String(t *testing.T) {
	var d UnixMilliDuration
	err := json.Unmarshal([]byte(`"5s"`), &d)
	if err != nil {
		t.Errorf("UnmarshalJSON(string) error = %v", err)
		return
	}
	if d != NewUnixMilliDuration(5*time.Second) {
		t.Errorf("UnmarshalJSON(string) = %v, want 5s", d)
	}
}

// Verify interfaces are implemented
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
