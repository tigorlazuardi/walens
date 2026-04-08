package source_types

import (
	"testing"

	"github.com/walens/walens/internal/sources"
	"github.com/walens/walens/internal/sources/booru"
)

func testRegistry() *sources.Registry {
	registry := sources.NewRegistry()
	registry.Register(booru.New())
	return registry
}

func TestServiceListSourceTypes(t *testing.T) {
	svc := NewService(testRegistry())
	items, err := svc.ListSourceTypes()
	if err != nil {
		t.Fatalf("ListSourceTypes() error = %v", err)
	}

	if len(items) == 0 {
		t.Error("expected at least one source type to be registered")
	}

	// Check booru is registered
	found := false
	for _, item := range items {
		if item.TypeName == "booru" {
			found = true
			if item.DisplayName != "Booru Image Board" {
				t.Errorf("expected display name 'Booru Image Board', got %q", item.DisplayName)
			}
			if item.DefaultLookupCount != 100 {
				t.Errorf("expected default lookup count 100, got %d", item.DefaultLookupCount)
			}
			if item.ParamSchema == nil {
				t.Error("expected param schema to be non-nil")
			}
			break
		}
	}
	if !found {
		t.Error("expected 'booru' source type to be registered")
	}
}

func TestServiceGetSourceType(t *testing.T) {
	svc := NewService(testRegistry())

	// Get existing source
	metadata, err := svc.GetSourceType("booru")
	if err != nil {
		t.Fatalf("GetSourceType() error = %v", err)
	}
	if metadata.TypeName != "booru" {
		t.Errorf("expected type name 'booru', got %q", metadata.TypeName)
	}
	if metadata.DisplayName != "Booru Image Board" {
		t.Errorf("expected display name 'Booru Image Board', got %q", metadata.DisplayName)
	}

	// Get non-existent source
	nonexistent, err := svc.GetSourceType("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent source type")
	}
	if nonexistent != nil {
		t.Error("expected nil metadata for non-existent source type")
	}
}

func TestServiceGetSourceTypeNotFound(t *testing.T) {
	svc := NewService(testRegistry())

	metadata, err := svc.GetSourceType("doesnotexist")
	if err == nil {
		t.Fatal("expected error for non-existent source type")
	}
	if metadata != nil {
		t.Error("expected nil for non-existent source type")
	}
}

func TestServiceListSourceTypesRegistryUnavailable(t *testing.T) {
	svc := NewService(nil)
	if _, err := svc.ListSourceTypes(); err == nil {
		t.Fatal("expected error when registry is nil")
	}
}
