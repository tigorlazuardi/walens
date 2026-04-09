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
//
// Lookup budget semantics:
//
//   - LookupCount specifies the upstream budget for how many items to check from
//     the source. This is NOT a guaranteed result count.
//   - Skipped items (non-image posts, filtered content, etc.) still count toward
//     the lookup budget.
//   - Deduplicated items (already known to the system) still count toward the
//     lookup budget.
//   - The actual number of ImageMetadata items yielded may be less than, equal
//     to, or even zero when LookupCount > 0, depending on source content and
//     filtering rules.
//   - Sources should request items from upstream until the budget is exhausted
//     or context is cancelled.
type FetchRequest struct {
	// Params is the source-specific configuration as raw JSON.
	// This is the user-configured params stored in the sources table.
	Params []byte

	// LookupCount is the upstream lookup budget. This determines how many
	// items the source should attempt to fetch/check from the upstream service.
	// The actual number of images returned may be less than this value.
	LookupCount int
}

// ImageMetadata represents metadata for a single image item from a source.
//
// All fields must be populated by the source implementation when yielding items
// from Fetch. The UniqueIdentifier field is required and is used to generate
// a stable unique ID via BuildUniqueID. SourceItemID is also required and
// should represent the external source's native identifier for the item.
type ImageMetadata struct {
	// UniqueIdentifier is a source-specific string that uniquely identifies
	// this image within the source. This is used as input to BuildUniqueID
	// to generate a stable global unique ID. Must be stable across fetches
	// for the same image.
	//
	// Required: Yes
	UniqueIdentifier string

	// PreviewURL is the URL to a smaller/preview version of the image,
	// typically used for gallery display. May be empty if no preview
	// is available.
	PreviewURL string

	// OriginURL is the URL to the original/full-resolution image file.
	// This is used as the source for downloading the canonical image.
	// May be empty if OriginURL is not available, though at least one
	// of PreviewURL or OriginURL should be populated.
	OriginURL string

	// SourceItemID is the external source's native unique identifier for
	// this item (e.g., the post ID from Gelbooru, the image hash from
	// Wallhaven, etc.). This is used as input to BuildUniqueID.
	//
	// Required: Yes
	SourceItemID string

	// OriginalID is the original/uploader-assigned identifier for the image.
	// This may be empty if the source does not provide such an ID.
	OriginalID string

	// Uploader is the username/name of the person who uploaded this image
	// to the source. May be empty.
	Uploader string

	// Artist is the artist/creator associated with this image. May be empty.
	Artist string

	// MimeType is the MIME type of the image (e.g., "image/jpeg", "image/png",
	// "image/webp"). Used for determining download and storage format.
	MimeType string

	// FileSizeBytes is the size of the original image file in bytes.
	// May be 0 if unknown.
	FileSizeBytes int64

	// Width is the pixel width of the image. May be 0 if unknown.
	Width int

	// Height is the pixel height of the image. May be 0 if unknown.
	Height int

	// AspectRatio is the width/height ratio (e.g., 1.777778 for 16:9).
	// May be 0 if Width or Height is unknown.
	AspectRatio float64

	// IsAdult indicates whether the image contains adult/adult content.
	// Used for filtering based on device preferences.
	IsAdult bool

	// Tags is a list of tags/categories associated with this image.
	// Used for filtering based on device tag preferences.
	Tags []string
}

// BuildUniqueIDInputs documents the fields from ImageMetadata that are
// required and used as inputs to BuildUniqueID for generating a stable
// unique identifier.
//
// The generated unique ID must:
//   - Be stable: same inputs always produce the same output
//   - Be unique: different inputs produce different outputs
//   - Be source-qualified: include source identity to avoid collisions
//     across different sources
//
// Required inputs for BuildUniqueID:
//   - SourceItemID: the external source's native identifier
//   - UniqueIdentifier: the source-specific unique string for this image
//
// Implementations should combine these inputs with the source type name
// to create a global unique identifier, typically via hashing.
type BuildUniqueIDInputs struct {
	_ struct{} // Exported struct marker
}

// Source defines the contract for a wallpaper source implementation.
// Implementations are registered in the global registry by type name.
//
// Iterator contract for Fetch:
//
// The iterator returned by Fetch must adhere to the following contract:
//
//   - Lazy evaluation: Items should be yielded progressively as they are
//     fetched from upstream, not buffered entirely before iteration begins.
//   - Context cancellation: The iterator must promptly honor context cancellation.
//     When ctx is cancelled, the iterator should stop fetching and return.
//   - Early stop support: Callers may stop iteration early without requiring
//     the iterator to drain remaining items or return an error.
//   - Error surfacing: Iteration errors should be yielded as they occur via
//     the error channel, not accumulated and returned at the end.
//
// BuildUniqueID contract:
//
// BuildUniqueID must generate a stable unique identifier that:
//   - Is deterministic: same ImageMetadata always produces the same ID
//   - Is globally unique: combines source identity with image identity
//   - Uses required inputs: SourceItemID and UniqueIdentifier
//
// It is called after Fetch yields each ImageMetadata item, typically to
// check for duplicates or generate storage paths.
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
	// This is used when no explicit LookupCount is configured.
	DefaultLookupCount() int

	// Fetch yields ImageMetadata items lazily from the source.
	//
	// The iterator yields items progressively. Callers may stop early.
	// Errors are yielded via the error channel as they occur.
	Fetch(ctx context.Context, req FetchRequest) iter.Seq2[ImageMetadata, error]

	// BuildUniqueID generates a stable unique identifier for an image item.
	//
	// Required inputs: item.SourceItemID and item.UniqueIdentifier.
	// The returned ID should be globally unique by incorporating
	// the source type name.
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
