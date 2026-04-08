// Package sources provides the source implementation registry and interfaces
// for wallpaper source plugins.
package sources

import (
	"context"
	"encoding/json"
	"iter"
	"sort"

	"github.com/danielgtaylor/huma/v2"
)

// FetchRequest is passed to Source.Fetch containing execution parameters.
type FetchRequest struct {
	Params      []byte
	LookupCount int
}

// ImageMetadata represents metadata for a single image item from a source.
type ImageMetadata struct {
	UniqueIdentifier string
	PreviewURL       string
	OriginURL        string
	SourceItemID     string
	OriginalID       string
	Uploader         string
	Artist           string
	MimeType         string
	FileSizeBytes    int64
	Width            int
	Height           int
	AspectRatio      float64
	IsAdult          bool
	Tags             []string
}

// Source defines the contract for a wallpaper source implementation.
// Implementations are registered in the global registry by type name.
type Source interface {
	// TypeName returns the unique implementation identifier, e.g. "booru".
	TypeName() string
	// DisplayName returns the human-readable name for UI display.
	DisplayName() string
	// ValidateParams validates the raw JSON params and returns an error if invalid.
	ValidateParams(raw json.RawMessage) error
	// ParamSchema returns the JSON Schema for the source params as a Huma Schema.
	ParamSchema() *huma.Schema
	// DefaultLookupCount returns the default upstream lookup budget.
	DefaultLookupCount() int
	// Fetch yields ImageMetadata items lazily from the source.
	Fetch(ctx context.Context, req FetchRequest) iter.Seq2[ImageMetadata, error]
	// BuildUniqueID generates a stable unique identifier for an image item.
	BuildUniqueID(item ImageMetadata) (string, error)
}

// Registry holds all registered source implementations.
type Registry struct {
	sources map[string]Source
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		sources: make(map[string]Source),
	}
}

// Register adds a source implementation to the registry.
// Panics if a source with the same TypeName is already registered.
func (r *Registry) Register(src Source) {
	name := src.TypeName()
	if _, exists := r.sources[name]; exists {
		panic("source already registered: " + name)
	}
	r.sources[name] = src
}

// Get returns a registered source by type name, or nil if not found.
func (r *Registry) Get(typeName string) Source {
	return r.sources[typeName]
}

// List returns all registered sources.
func (r *Registry) List() []Source {
	result := make([]Source, 0, len(r.sources))
	for _, src := range r.sources {
		result = append(result, src)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TypeName() < result[j].TypeName()
	})
	return result
}

// Has returns true if a source with the given type name is registered.
func (r *Registry) Has(typeName string) bool {
	_, ok := r.sources[typeName]
	return ok
}
