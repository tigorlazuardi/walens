package dbtypes

import (
	"errors"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/table"
)

func TestSort_IsValid(t *testing.T) {
	// Create a mock column list for testing
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("name"),
		sqlite.StringColumn("created_at"),
		sqlite.StringColumn("updated_at"),
	}

	tests := []struct {
		name     string
		sort     *Sort
		expected bool
	}{
		{
			name:     "nil sort returns true (no sorting applied)",
			sort:     nil,
			expected: true,
		},
		{
			name: "valid field 'id'",
			sort: &Sort{
				Field: "id",
				Order: strPtr("asc"),
			},
			expected: true,
		},
		{
			name: "valid field 'name'",
			sort: &Sort{
				Field: "name",
				Order: strPtr("desc"),
			},
			expected: true,
		},
		{
			name: "valid field 'created_at'",
			sort: &Sort{
				Field: "created_at",
				Order: nil, // nil order should still be valid
			},
			expected: true,
		},
		{
			name: "invalid field 'nonexistent'",
			sort: &Sort{
				Field: "nonexistent",
				Order: strPtr("asc"),
			},
			expected: false,
		},
		{
			name: "invalid field empty string",
			sort: &Sort{
				Field: "",
				Order: strPtr("asc"),
			},
			expected: false,
		},
		{
			name: "case-sensitive field name",
			sort: &Sort{
				Field: "ID", // uppercase should not match lowercase 'id'
				Order: strPtr("asc"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sort.IsValid(columns)
			if result != tt.expected {
				t.Errorf("IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSortList_BuildOrderByClause_EmptyList(t *testing.T) {
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("name"),
	}

	// Empty SortList should return empty clauses without error
	var list SortList = []Sort{}
	clauses, err := list.BuildOrderByClause(columns)
	if err != nil {
		t.Errorf("expected no error for empty list, got: %v", err)
	}
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses, got %d", len(clauses))
	}
}

func TestSortList_BuildOrderByClause_NilList(t *testing.T) {
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("name"),
	}

	// Nil SortList should behave like empty list
	var list SortList = nil
	clauses, err := list.BuildOrderByClause(columns)
	if err != nil {
		t.Errorf("expected no error for nil list, got: %v", err)
	}
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses, got %d", len(clauses))
	}
}

func TestSortList_BuildOrderByClause_SingleValidSort(t *testing.T) {
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("name"),
	}

	tests := []struct {
		name string
		sort Sort
	}{
		{
			name: "single sort asc",
			sort: Sort{
				Field: "id",
				Order: strPtr("asc"),
			},
		},
		{
			name: "single sort desc",
			sort: Sort{
				Field: "name",
				Order: strPtr("desc"),
			},
		},
		{
			name: "default order is asc when nil",
			sort: Sort{
				Field: "id",
				Order: nil,
			},
		},
		{
			name: "default order is asc when empty string",
			sort: Sort{
				Field: "id",
				Order: strPtr(""),
			},
		},
		{
			name: "case insensitive order value ASC",
			sort: Sort{
				Field: "id",
				Order: strPtr("ASC"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := SortList{tt.sort}
			clauses, err := list.BuildOrderByClause(columns)
			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if len(clauses) != 1 {
				t.Errorf("expected 1 clause, got %d", len(clauses))
			}
			// Note: OrderByClause is an internal jet type, we verify by checking
			// that clauses were created without error and have correct count
		})
	}
}

func TestSortList_BuildOrderByClause_MultipleValidSorts(t *testing.T) {
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("name"),
		sqlite.StringColumn("created_at"),
	}

	list := SortList{
		{Field: "created_at", Order: strPtr("desc")},
		{Field: "name", Order: strPtr("asc")},
		{Field: "id", Order: nil},
	}

	clauses, err := list.BuildOrderByClause(columns)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(clauses) != 3 {
		t.Errorf("expected 3 clauses, got %d", len(clauses))
	}
	// Verify that we got the expected number of clauses for valid sorts
}

func TestSortList_BuildOrderByClause_InvalidField(t *testing.T) {
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("name"),
	}

	list := SortList{
		{Field: "nonexistent", Order: strPtr("asc")},
	}

	clauses, err := list.BuildOrderByClause(columns)

	// Should return huma.Error400BadRequest
	if err == nil {
		t.Error("expected error for invalid field, got nil")
	}

	// Check that it's a huma error with 400 status
	var humaErr *huma.ErrorModel
	if !errors.As(err, &humaErr) {
		t.Error("expected huma.ErrorModel")
	} else if humaErr.Status != 400 {
		t.Errorf("expected status 400, got %d", humaErr.Status)
	}

	// Should still return empty clauses
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses for invalid field, got %d", len(clauses))
	}
}

func TestSortList_BuildOrderByClause_MixedValidAndInvalid(t *testing.T) {
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("name"),
	}

	list := SortList{
		{Field: "id", Order: strPtr("asc")},        // valid
		{Field: "invalid1", Order: strPtr("desc")}, // invalid
		{Field: "name", Order: strPtr("asc")},      // valid
		{Field: "invalid2", Order: nil},            // invalid
	}

	clauses, err := list.BuildOrderByClause(columns)

	// Should return error because of invalid fields
	if err == nil {
		t.Error("expected error for mixed valid/invalid fields, got nil")
	}

	// Check that it's a huma error
	var humaErr *huma.ErrorModel
	if !errors.As(err, &humaErr) {
		t.Error("expected huma.ErrorModel")
	}

	// Should return clauses for valid fields only
	if len(clauses) != 2 {
		t.Errorf("expected 2 clauses (valid only), got %d", len(clauses))
	}
	// Valid clauses are returned even when there are errors
}

func TestSortList_BuildOrderByClause_AllInvalidFields(t *testing.T) {
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("name"),
	}

	list := SortList{
		{Field: "invalid1", Order: strPtr("asc")},
		{Field: "invalid2", Order: strPtr("desc")},
	}

	clauses, err := list.BuildOrderByClause(columns)

	if err == nil {
		t.Error("expected error for all invalid fields, got nil")
	}

	var humaErr *huma.ErrorModel
	if !errors.As(err, &humaErr) {
		t.Error("expected huma.ErrorModel")
	} else if humaErr.Status != 400 {
		t.Errorf("expected status 400, got %d", humaErr.Status)
	}

	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses, got %d", len(clauses))
	}
}

func TestSortList_BuildOrderByClause_EmptyFieldName(t *testing.T) {
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
	}

	list := SortList{
		{Field: "", Order: strPtr("asc")},
	}

	clauses, err := list.BuildOrderByClause(columns)

	if err == nil {
		t.Error("expected error for empty field name, got nil")
	}

	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses, got %d", len(clauses))
	}
}

func TestSortList_BuildOrderByClause_CaseSensitivity(t *testing.T) {
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("Name"), // uppercase N
	}

	tests := []struct {
		name          string
		field         string
		shouldBeValid bool
	}{
		{
			name:          "lowercase id matches lowercase column",
			field:         "id",
			shouldBeValid: true,
		},
		{
			name:          "uppercase ID does not match lowercase id",
			field:         "ID",
			shouldBeValid: false,
		},
		{
			name:          "lowercase name does not match Name",
			field:         "name",
			shouldBeValid: false,
		},
		{
			name:          "exact case Name matches",
			field:         "Name",
			shouldBeValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := SortList{{Field: tt.field, Order: strPtr("asc")}}
			clauses, err := list.BuildOrderByClause(columns)

			if tt.shouldBeValid {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				if len(clauses) != 1 {
					t.Errorf("expected 1 clause, got %d", len(clauses))
				}
			} else {
				if err == nil {
					t.Error("expected error for case mismatch, got nil")
				}
			}
		})
	}
}

func TestSortList_BuildOrderByClause_WithGeneratedTable(t *testing.T) {
	// Test using actual generated table columns (Devices table)
	// This ensures our SortList works with real generated code
	columns := table.Devices.AllColumns

	tests := []struct {
		name        string
		sorts       SortList
		expectErr   bool
		expectCount int
	}{
		{
			name: "sort by created_at desc (common use case)",
			sorts: SortList{
				{Field: "created_at", Order: strPtr("desc")},
			},
			expectErr:   false,
			expectCount: 1,
		},
		{
			name: "sort by name asc then created_at desc",
			sorts: SortList{
				{Field: "name", Order: strPtr("asc")},
				{Field: "created_at", Order: strPtr("desc")},
			},
			expectErr:   false,
			expectCount: 2,
		},
		{
			name: "sort by slug with default order",
			sorts: SortList{
				{Field: "slug", Order: nil},
			},
			expectErr:   false,
			expectCount: 1,
		},
		{
			name: "sort by non-existent field",
			sorts: SortList{
				{Field: "nonexistent_field", Order: strPtr("asc")},
			},
			expectErr:   true,
			expectCount: 0,
		},
		{
			name: "mixed valid and invalid fields",
			sorts: SortList{
				{Field: "name", Order: strPtr("asc")},
				{Field: "invalid", Order: strPtr("desc")},
				{Field: "screen_width", Order: strPtr("asc")},
			},
			expectErr:   true,
			expectCount: 2, // valid fields still returned
		},
		{
			name: "multiple valid fields all device columns",
			sorts: SortList{
				{Field: "id", Order: strPtr("asc")},
				{Field: "name", Order: strPtr("asc")},
				{Field: "slug", Order: strPtr("asc")},
				{Field: "screen_width", Order: strPtr("desc")},
				{Field: "screen_height", Order: strPtr("desc")},
				{Field: "created_at", Order: strPtr("desc")},
				{Field: "updated_at", Order: strPtr("desc")},
			},
			expectErr:   false,
			expectCount: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clauses, err := tt.sorts.BuildOrderByClause(columns)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				// Verify it's a huma error
				var humaErr *huma.ErrorModel
				if !errors.As(err, &humaErr) {
					t.Error("expected huma.ErrorModel for validation error")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			if len(clauses) != tt.expectCount {
				t.Errorf("expected %d clauses, got %d", tt.expectCount, len(clauses))
			}
		})
	}
}

func TestSortList_BuildOrderByClause_EmptySortListWithRealTable(t *testing.T) {
	// Test that empty SortList works with real table columns
	columns := sqlite.ColumnList{
		sqlite.StringColumn("id"),
		sqlite.StringColumn("name"),
		sqlite.StringColumn("created_at"),
	}

	var list SortList = []Sort{}
	clauses, err := list.BuildOrderByClause(columns)
	if err != nil {
		t.Errorf("expected no error for empty list with real table, got: %v", err)
	}
	if len(clauses) != 0 {
		t.Errorf("expected 0 clauses, got %d", len(clauses))
	}
}

// Helper function
func strPtr(s string) *string {
	return &s
}
