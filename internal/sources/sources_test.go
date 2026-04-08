package sources

import (
	"context"
	"encoding/json"
	"iter"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

// mockSource is a test source implementation.
type mockSource struct {
	name         string
	displayName  string
	defaultCount int
	paramSchema  *huma.Schema
	validateErr  error
}

func (s *mockSource) TypeName() string {
	return s.name
}

func (s *mockSource) DisplayName() string {
	return s.displayName
}

func (s *mockSource) ValidateParams(raw json.RawMessage) error {
	return s.validateErr
}

func (s *mockSource) ParamSchema() *huma.Schema {
	return s.paramSchema
}

func (s *mockSource) DefaultLookupCount() int {
	return s.defaultCount
}

func (s *mockSource) Fetch(ctx context.Context, req FetchRequest) iter.Seq2[ImageMetadata, error] {
	return func(yield func(ImageMetadata, error) bool) {}
}

func (s *mockSource) BuildUniqueID(item ImageMetadata) (string, error) {
	return "mock:" + item.SourceItemID, nil
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	src := &mockSource{
		name:         "test",
		displayName:  "Test Source",
		defaultCount: 50,
	}

	registry.Register(src)

	if !registry.Has("test") {
		t.Error("expected source 'test' to be registered")
	}
}

func TestRegistryRegisterPanics(t *testing.T) {
	registry := NewRegistry()

	src1 := &mockSource{name: "dup", displayName: "First", defaultCount: 10}
	src2 := &mockSource{name: "dup", displayName: "Second", defaultCount: 20}

	registry.Register(src1)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration")
		}
	}()

	registry.Register(src2)
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry()

	src := &mockSource{name: "gettest", displayName: "Get Test", defaultCount: 25}
	registry.Register(src)

	got := registry.Get("gettest")
	if got == nil {
		t.Fatal("expected to get registered source")
	}
	if got.DisplayName() != "Get Test" {
		t.Errorf("expected display name 'Get Test', got %q", got.DisplayName())
	}

	// Non-existent
	if registry.Get("nonexistent") != nil {
		t.Error("expected nil for non-existent source")
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	src1 := &mockSource{name: "list1", displayName: "List 1", defaultCount: 10}
	src2 := &mockSource{name: "list2", displayName: "List 2", defaultCount: 20}

	registry.Register(src1)
	registry.Register(src2)

	list := registry.List()
	if len(list) != 2 {
		t.Errorf("expected 2 sources, got %d", len(list))
	}
}

func TestRegistryHas(t *testing.T) {
	registry := NewRegistry()

	src := &mockSource{name: "has", displayName: "Has Test", defaultCount: 10}
	registry.Register(src)

	if !registry.Has("has") {
		t.Error("expected Has('has') to be true")
	}
	if registry.Has("nope") {
		t.Error("expected Has('nope') to be false")
	}
}

func TestRegistryListSortedByTypeName(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockSource{name: "z-last", displayName: "Z", defaultCount: 10})
	registry.Register(&mockSource{name: "a-first", displayName: "A", defaultCount: 20})

	list := registry.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(list))
	}
	if list[0].TypeName() != "a-first" || list[1].TypeName() != "z-last" {
		t.Fatalf("expected sorted source type names, got %q then %q", list[0].TypeName(), list[1].TypeName())
	}
}
