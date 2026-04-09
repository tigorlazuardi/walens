package dbtypes

import (
	"testing"

	"github.com/google/uuid"
)

func TestCursorPaginationRequest_GetLimitOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		req      *CursorPaginationRequest
		def      int64
		max      int64
		expected int64
	}{
		{
			name:     "nil request returns default",
			req:      nil,
			def:      20,
			max:      100,
			expected: 20,
		},
		{
			name:     "nil limit returns default",
			req:      &CursorPaginationRequest{},
			def:      20,
			max:      100,
			expected: 20,
		},
		{
			name: "zero limit returns default",
			req: &CursorPaginationRequest{
				Limit: intPtr(0),
			},
			def:      20,
			max:      100,
			expected: 20,
		},
		{
			name: "negative limit returns default",
			req: &CursorPaginationRequest{
				Limit: intPtr(-5),
			},
			def:      20,
			max:      100,
			expected: 20,
		},
		{
			name: "positive limit within max",
			req: &CursorPaginationRequest{
				Limit: intPtr(50),
			},
			def:      20,
			max:      100,
			expected: 50,
		},
		{
			name: "limit exceeding max returns max",
			req: &CursorPaginationRequest{
				Limit: intPtr(150),
			},
			def:      20,
			max:      100,
			expected: 100,
		},
		{
			name: "limit exactly at max",
			req: &CursorPaginationRequest{
				Limit: intPtr(100),
			},
			def:      20,
			max:      100,
			expected: 100,
		},
		{
			name: "limit of 1",
			req: &CursorPaginationRequest{
				Limit: intPtr(1),
			},
			def:      20,
			max:      100,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.GetLimitOrDefault(tt.def, tt.max)
			if result != tt.expected {
				t.Errorf("GetLimitOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCursorPaginationRequest_NextToken(t *testing.T) {
	tests := []struct {
		name     string
		req      *CursorPaginationRequest
		expected string
	}{
		{
			name:     "nil request returns empty",
			req:      nil,
			expected: "",
		},
		{
			name:     "nil next returns empty",
			req:      &CursorPaginationRequest{},
			expected: "",
		},
		{
			name: "valid next token",
			req: &CursorPaginationRequest{
				Next: mustNewUUIDV7(t),
			},
			expected: "", // Will be a valid UUID string, checked separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.NextToken()
			if tt.name == "valid next token" {
				// For valid UUID, result should be non-empty and parseable
				if result == "" {
					t.Error("NextToken() should return non-empty string for valid UUID")
				}
				_, err := uuid.Parse(result)
				if err != nil {
					t.Errorf("NextToken() returned invalid UUID: %v", err)
				}
			} else {
				if result != tt.expected {
					t.Errorf("NextToken() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestCursorPaginationRequest_PrevToken(t *testing.T) {
	tests := []struct {
		name     string
		req      *CursorPaginationRequest
		expected string
	}{
		{
			name:     "nil request returns empty",
			req:      nil,
			expected: "",
		},
		{
			name:     "nil prev returns empty",
			req:      &CursorPaginationRequest{},
			expected: "",
		},
		{
			name: "valid prev token",
			req: &CursorPaginationRequest{
				Prev: mustNewUUIDV7(t),
			},
			expected: "", // Will be a valid UUID string, checked separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.PrevToken()
			if tt.name == "valid prev token" {
				// For valid UUID, result should be non-empty and parseable
				if result == "" {
					t.Error("PrevToken() should return non-empty string for valid UUID")
				}
				_, err := uuid.Parse(result)
				if err != nil {
					t.Errorf("PrevToken() returned invalid UUID: %v", err)
				}
			} else {
				if result != tt.expected {
					t.Errorf("PrevToken() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestCursorPaginationRequest_GetOffset(t *testing.T) {
	tests := []struct {
		name     string
		req      *CursorPaginationRequest
		expected int64
	}{
		{
			name:     "nil request returns 0",
			req:      nil,
			expected: 0,
		},
		{
			name:     "nil offset returns 0",
			req:      &CursorPaginationRequest{},
			expected: 0,
		},
		{
			name: "positive offset",
			req: &CursorPaginationRequest{
				Offset: intPtr(50),
			},
			expected: 50,
		},
		{
			name: "zero offset",
			req: &CursorPaginationRequest{
				Offset: intPtr(0),
			},
			expected: 0,
		},
		{
			name: "negative offset returns 0",
			req: &CursorPaginationRequest{
				Offset: intPtr(-10),
			},
			expected: 0,
		},
		{
			name: "large offset",
			req: &CursorPaginationRequest{
				Offset: intPtr(10000),
			},
			expected: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.req.GetOffset()
			if result != tt.expected {
				t.Errorf("GetOffset() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCursorPaginationRequest_FullConfiguration(t *testing.T) {
	// Test a fully configured pagination request
	nextUUID := mustNewUUIDV7(t)
	prevUUID := mustNewUUIDV7(t)
	limit := 25
	offset := 100

	req := &CursorPaginationRequest{
		Next:   nextUUID,
		Prev:   prevUUID,
		Limit:  &limit,
		Offset: &offset,
	}

	// Test all methods
	if req.NextToken() != nextUUID.String() {
		t.Error("NextToken() should match Next UUID")
	}

	if req.PrevToken() != prevUUID.String() {
		t.Error("PrevToken() should match Prev UUID")
	}

	if req.GetLimitOrDefault(20, 50) != 25 {
		t.Error("GetLimitOrDefault() should return 25")
	}

	if req.GetOffset() != 100 {
		t.Error("GetOffset() should return 100")
	}
}

func TestCursorPaginationRequest_NextPrevPrecedence(t *testing.T) {
	// Test that both Next and Prev can be set, and they work independently
	nextUUID := mustNewUUIDV7(t)
	prevUUID := mustNewUUIDV7(t)

	req := &CursorPaginationRequest{
		Next: nextUUID,
		Prev: prevUUID,
	}

	nextToken := req.NextToken()
	prevToken := req.PrevToken()

	if nextToken == "" {
		t.Error("NextToken() should not be empty when Next is set")
	}

	if prevToken == "" {
		t.Error("PrevToken() should not be empty when Prev is set")
	}

	if nextToken == prevToken {
		t.Error("Next and Prev tokens should be different")
	}
}

// Helper functions

func intPtr(i int) *int {
	return &i
}

func mustNewUUIDV7(t *testing.T) *UUID {
	t.Helper()
	u, err := NewUUIDV7()
	if err != nil {
		t.Fatalf("failed to create UUIDv7: %v", err)
	}
	return &u
}
