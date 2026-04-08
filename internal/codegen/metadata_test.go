package codegen

import (
	"testing"

	"github.com/go-jet/jet/v2/generator/metadata"
)

func TestIsUUIDColumn(t *testing.T) {
	tests := []struct {
		name       string
		columnName string
		dbType     string
		expected   bool
	}{
		{"id column text", "id", "text", true},
		{"source_id column text", "source_id", "text", true},
		{"device_id column text", "device_id", "text", true},
		{"image_id column varchar", "image_id", "varchar", true},
		{"tag_id column char", "tag_id", "char", true},
		{"external identifier", "source_identifier", "text", false},
		{"original_identifier", "original_identifier", "text", false},
		{"random text column", "name", "text", false},
		{"integer column", "some_id", "integer", false},
		{"job_id text", "job_id", "text", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUUIDColumn(tt.columnName, tt.dbType)
			if result != tt.expected {
				t.Errorf("IsUUIDColumn(%q, %q) = %v, want %v", tt.columnName, tt.dbType, result, tt.expected)
			}
		})
	}
}

func TestIsUnixMilliTimestampColumn(t *testing.T) {
	tests := []struct {
		name       string
		columnName string
		dbType     string
		expected   bool
	}{
		// Explicit known timestamp columns
		{"created_at integer", "created_at", "integer", true},
		{"updated_at integer", "updated_at", "integer", true},
		{"started_at integer", "started_at", "integer", true},
		{"finished_at integer", "finished_at", "integer", true},
		{"run_after integer", "run_after", "integer", true},
		{"deleted_at integer", "deleted_at", "integer", true},
		{"archived_at integer", "archived_at", "integer", true},
		{"last_run_at integer", "last_run_at", "integer", true},
		{"next_run_at integer", "next_run_at", "integer", true},
		{"claimed_at integer", "claimed_at", "integer", true},
		{"locked_at integer", "locked_at", "integer", true},
		// Non-integer types should be false
		{"created_at text", "created_at", "text", false},
		{"created_at varchar", "created_at", "varchar", false},
		// NOT in known set, should be false even with _at suffix
		{"last_check_at integer", "last_check_at", "integer", false},
		{"status_changed_at integer", "status_changed_at", "integer", false},
		{"cost_at integer", "cost_at", "integer", false},
		{"rate_at integer", "rate_at", "integer", false},
		// Not in known set
		{"count integer", "count", "integer", false},
		{"timestamp_col integer", "timestamp_col", "integer", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUnixMilliTimestampColumn(tt.columnName, tt.dbType)
			if result != tt.expected {
				t.Errorf("IsUnixMilliTimestampColumn(%q, %q) = %v, want %v", tt.columnName, tt.dbType, result, tt.expected)
			}
		})
	}
}

func TestIsDurationColumn(t *testing.T) {
	tests := []struct {
		name       string
		columnName string
		dbType     string
		expected   bool
	}{
		// Explicit known duration columns
		{"duration_ms integer", "duration_ms", "integer", true},
		{"elapsed_ms integer", "elapsed_ms", "integer", true},
		{"processing_ms integer", "processing_ms", "integer", true},
		{"latency_ms integer", "latency_ms", "integer", true},
		{"timeout_ms integer", "timeout_ms", "integer", true},
		// Different integer types
		{"duration_ms int", "duration_ms", "int", true},
		{"duration_ms int64", "duration_ms", "int64", true},
		// NOT in known set - should be false even with _ms suffix
		{"some_ms integer", "some_ms", "integer", false},
		{"count_ms integer", "count_ms", "integer", false},
		{"max_ms integer", "max_ms", "integer", false},
		// Wrong type
		{"duration_ms text", "duration_ms", "text", false},
		// Not _ms suffix
		{"duration_seconds integer", "duration_seconds", "integer", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDurationColumn(tt.columnName, tt.dbType)
			if result != tt.expected {
				t.Errorf("IsDurationColumn(%q, %q) = %v, want %v", tt.columnName, tt.dbType, result, tt.expected)
			}
		})
	}
}

func TestIsBoolIntColumn(t *testing.T) {
	tests := []struct {
		name       string
		columnName string
		dbType     string
		expected   bool
	}{
		{"is_enabled integer", "is_enabled", "integer", true},
		{"is_adult_allowed integer", "is_adult_allowed", "integer", true},
		{"is_active int", "is_active", "int", true},
		{"is_deleted int64", "is_deleted", "int64", true},
		{"is_valid integer", "is_valid", "integer", true},
		{"enabled text", "enabled", "text", false},
		{"admin_flag integer", "admin_flag", "integer", false},
		{"has_is_prefix text", "has_is", "text", false},
		{"is_something varchar", "is_something", "varchar", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBoolIntColumn(tt.columnName, tt.dbType)
			if result != tt.expected {
				t.Errorf("IsBoolIntColumn(%q, %q) = %v, want %v", tt.columnName, tt.dbType, result, tt.expected)
			}
		})
	}
}

func TestIsRawJSONColumn(t *testing.T) {
	tests := []struct {
		name       string
		columnName string
		dbType     string
		expected   bool
	}{
		// Explicit known JSON columns
		{"params text", "params", "text", true},
		{"json_meta text", "json_meta", "text", true},
		{"json_input text", "json_input", "text", true},
		{"json_result text", "json_result", "text", true},
		{"metadata text", "metadata", "text", true},
		{"config text", "config", "text", true},
		{"settings text", "settings", "text", true},
		{"payload text", "payload", "text", true},
		// json_ prefix
		{"json_something varchar", "json_something", "varchar", true},
		{"json_data char", "json_data", "char", true},
		// NOT in known set
		{"data text", "data", "text", false},
		{"description text", "description", "text", false},
		// Wrong type
		{"params integer", "params", "integer", false},
		{"metadata blob", "metadata", "blob", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRawJSONColumn(tt.columnName, tt.dbType)
			if result != tt.expected {
				t.Errorf("IsRawJSONColumn(%q, %q) = %v, want %v", tt.columnName, tt.dbType, result, tt.expected)
			}
		})
	}
}

func TestClassifyColumn(t *testing.T) {
	tests := []struct {
		name             string
		columnName       string
		dbType           string
		expectedGoType   string
		expectedNullable bool
	}{
		// UUID
		{"id column", "id", "text", TypeUUID, false},
		{"source_id column", "source_id", "text", TypeUUID, false},
		// Timestamp
		{"created_at column", "created_at", "integer", TypeUnixMilliTime, false},
		{"updated_at column", "updated_at", "integer", TypeUnixMilliTime, false},
		// Duration
		{"duration_ms column", "duration_ms", "integer", TypeUnixMilliDuration, false},
		// Bool
		{"is_enabled column", "is_enabled", "integer", TypeBoolInt, false},
		// JSON
		{"json_meta column", "json_meta", "text", TypeRawJSON, false},
		{"params column", "params", "text", TypeRawJSON, false},
		// Defaults
		{"name column", "name", "text", TypeString, false},
		{"count column", "count", "integer", TypeInt64, false},
		{"price column", "price", "real", TypeFloat64, false},
		{"data column", "data", "blob", TypeBytes, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goType, nullable := ClassifyColumn(tt.columnName, tt.dbType)
			if goType != tt.expectedGoType {
				t.Errorf("ClassifyColumn(%q, %q) goType = %v, want %v", tt.columnName, tt.dbType, goType, tt.expectedGoType)
			}
			if nullable != tt.expectedNullable {
				t.Errorf("ClassifyColumn(%q, %q) nullable = %v, want %v", tt.columnName, tt.dbType, nullable, tt.expectedNullable)
			}
		})
	}
}

func TestToGoName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"created_at", "CreatedAt"},
		{"updated_at", "UpdatedAt"},
		{"is_enabled", "IsEnabled"},
		{"source_id", "SourceID"},
		{"image_id", "ImageID"},
		{"json_meta", "JsonMeta"},
		{"duration_ms", "DurationMs"},
		{"device", "Device"},
		{"id", "ID"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToGoName(tt.input)
			if result != tt.expected {
				t.Errorf("ToGoName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToGoType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{TypeUUID, "dbtypes.UUID"},
		{TypeBoolInt, "dbtypes.BoolInt"},
		{TypeUnixMilliTime, "dbtypes.UnixMilliTime"},
		{TypeUnixMilliDuration, "dbtypes.UnixMilliDuration"},
		{TypeRawJSON, "dbtypes.RawJSON"},
		{TypeString, "string"},
		{TypeInt64, "int64"},
		{TypeFloat64, "float64"},
		{TypeBytes, "[]byte"},
		{"unknown", "any"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToGoType(tt.input)
			if result != tt.expected {
				t.Errorf("ToGoType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDocTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Some description", `doc:"Some description"`},
		{"", ""},
		{"Has \"quotes\"", `doc:"Has \"quotes\""`},
		{"Multiple words here", `doc:"Multiple words here"`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := DocTag(tt.input)
			if result != tt.expected {
				t.Errorf("DocTag(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildColumnTags(t *testing.T) {
	col := ColumnDoc{
		Description: "The device name",
		IsRequired:  true,
	}

	result := BuildColumnTags("name", TypeString, col)
	if result == "" {
		t.Error("BuildColumnTags returned empty string")
	}
}

func TestValidateIdentifierName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"id", "id", false},
		{"source_id", "source_id", false},
		{"device_id", "device_id", false},
		{"image_id", "image_id", false},
		{"job_id", "job_id", false},
		{"external identifier", "source_identifier", false},
		{"original_identifier", "original_identifier", false},
		{"random_id", "random_id", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIdentifierName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIdentifierName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestGetColumnDoc(t *testing.T) {
	doc := GetColumnDoc("devices", "name")
	if doc.Description == "" {
		t.Error("GetColumnDoc for devices.name should return non-empty description")
	}
	if !doc.IsRequired {
		t.Error("devices.name should be required")
	}

	// Non-existent
	doc = GetColumnDoc("devices", "nonexistent")
	if doc.Description != "" {
		t.Error("GetColumnDoc for nonexistent column should return empty")
	}

	// Non-existent table
	doc = GetColumnDoc("nonexistent_table", "name")
	if doc.Description != "" {
		t.Error("GetColumnDoc for nonexistent table should return empty")
	}
}

func TestGetTableDoc(t *testing.T) {
	table := GetTableDoc("devices")
	if table == nil {
		t.Fatal("GetTableDoc for devices should return non-nil")
	}
	if table.Description == "" {
		t.Error("devices table should have description")
	}

	table = GetTableDoc("nonexistent")
	if table != nil {
		t.Error("GetTableDoc for nonexistent table should return nil")
	}
}

func TestValidateSchemaDocs(t *testing.T) {
	// Valid case: all columns documented
	err := ValidateSchemaDocs("configs", []string{"id", "value", "updated_at"})
	if err != nil {
		t.Errorf("ValidateSchemaDocs for configs should pass: %v", err)
	}

	// Missing column
	err = ValidateSchemaDocs("devices", []string{"id", "name", "slug", "nonexistent_col"})
	if err == nil {
		t.Error("ValidateSchemaDocs should fail for missing column documentation")
	}

	// Non-existent table
	err = ValidateSchemaDocs("nonexistent_table", []string{"id"})
	if err == nil {
		t.Error("ValidateSchemaDocs should fail for non-existent table")
	}
}

func TestWalensSchemaDocsCoverage(t *testing.T) {
	// This test verifies that the business schema columns are documented.
	// If this test fails, you need to add documentation to WalensSchemaDocs.

	businessTables := []string{
		"configs", "devices", "sources", "source_schedules",
		"device_source_subscriptions", "images", "tags", "image_tags",
		"image_assignments", "image_locations", "image_thumbnails",
		"image_blacklists", "jobs",
	}

	for _, tableName := range businessTables {
		t.Run(tableName, func(t *testing.T) {
			table, ok := WalensSchemaDocs[tableName]
			if !ok {
				t.Errorf("Table %q is missing from WalensSchemaDocs", tableName)
				return
			}
			if table.Description == "" {
				t.Errorf("Table %q has no description", tableName)
			}
			if len(table.Columns) == 0 {
				t.Errorf("Table %q has no columns documented", tableName)
			}
		})
	}
}

func TestClassifyColumnForGenerator(t *testing.T) {
	tests := []struct {
		name            string
		tableName       string
		columnName      string
		isNullable      bool
		dbType          string
		expectedGoType  string
		expectedWrapped string
		expectedPointer bool
	}{
		// UUID columns
		{"id column text", "devices", "id", false, "text", TypeUUID, "dbtypes.UUID", false},
		{"source_id text nullable", "images", "source_id", true, "text", TypeUUID, "dbtypes.UUID", true},
		// Timestamp columns
		{"created_at integer", "devices", "created_at", false, "integer", TypeUnixMilliTime, "dbtypes.UnixMilliTime", false},
		{"updated_at integer", "devices", "updated_at", false, "integer", TypeUnixMilliTime, "dbtypes.UnixMilliTime", false},
		// Duration columns
		{"duration_ms integer", "jobs", "duration_ms", false, "integer", TypeUnixMilliDuration, "dbtypes.UnixMilliDuration", false},
		// Bool columns
		{"is_enabled integer", "devices", "is_enabled", false, "integer", TypeBoolInt, "dbtypes.BoolInt", false},
		// JSON columns
		{"json_meta text", "images", "json_meta", false, "text", TypeRawJSON, "dbtypes.RawJSON", false},
		{"configs value text", "configs", "value", false, "text", TypeRawJSON, "dbtypes.RawJSON", false},
		// Default types
		{"name column text", "devices", "name", false, "text", TypeString, "string", false},
		{"count integer", "devices", "screen_width", false, "integer", TypeInt64, "int64", false},
		// External identifiers should NOT be UUID
		{"unique_identifier text", "images", "unique_identifier", false, "text", TypeString, "string", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goType, wrappedType, needsPointer := ClassifyColumnForGenerator(tt.tableName, tt.columnName, tt.isNullable, tt.dbType)
			if goType != tt.expectedGoType {
				t.Errorf("ClassifyColumnForGenerator(%q, %q, %v, %q) goType = %v, want %v",
					tt.tableName, tt.columnName, tt.isNullable, tt.dbType, goType, tt.expectedGoType)
			}
			if wrappedType != tt.expectedWrapped {
				t.Errorf("ClassifyColumnForGenerator(%q, %q, %v, %q) wrappedType = %v, want %v",
					tt.tableName, tt.columnName, tt.isNullable, tt.dbType, wrappedType, tt.expectedWrapped)
			}
			if needsPointer != tt.expectedPointer {
				t.Errorf("ClassifyColumnForGenerator(%q, %q, %v, %q) needsPointer = %v, want %v",
					tt.tableName, tt.columnName, tt.isNullable, tt.dbType, needsPointer, tt.expectedPointer)
			}
		})
	}
}

func TestCreateDocTag(t *testing.T) {
	tests := []struct {
		tableName    string
		columnName   string
		expectEmpty  bool
		expectPrefix string
	}{
		{"devices", "name", false, `doc:"Human-readable device name"`},
		{"devices", "id", false, `doc:"Unique device identifier (UUIDv7)"`},
		{"images", "created_at", false, `doc:"Image creation timestamp"`},
		{"devices", "nonexistent", true, ""},
		{"nonexistent_table", "name", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.tableName+"."+tt.columnName, func(t *testing.T) {
			result := createDocTag(
				metadata.Table{Name: tt.tableName},
				metadata.Column{Name: tt.columnName},
			)
			if tt.expectEmpty && result != "" {
				t.Errorf("createDocTag(%q, %q) = %q, want empty", tt.tableName, tt.columnName, result)
			}
			if !tt.expectEmpty && result == "" {
				t.Errorf("createDocTag(%q, %q) = empty, want non-empty", tt.tableName, tt.columnName)
			}
			if !tt.expectEmpty && tt.expectPrefix != "" && result != tt.expectPrefix {
				t.Errorf("createDocTag(%q, %q) = %q, want %q", tt.tableName, tt.columnName, result, tt.expectPrefix)
			}
		})
	}
}
