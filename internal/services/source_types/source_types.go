package source_types

import (
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/sources"
)

// ErrRegistryUnavailable is returned when the source registry is not configured.
var ErrRegistryUnavailable = errors.New("source registry unavailable")

// ErrSourceTypeNotFound is returned when the requested source type is not registered.
var ErrSourceTypeNotFound = errors.New("source type not found")

// SourceTypeMetadata contains the metadata for a registered source type.
type SourceTypeMetadata struct {
	TypeName           string       `json:"type_name" doc:"Unique implementation identifier."`
	DisplayName        string       `json:"display_name" doc:"Human-readable name for UI."`
	DefaultLookupCount int          `json:"default_lookup_count" doc:"Default upstream lookup budget per run."`
	ParamSchema        *huma.Schema `json:"param_schema" doc:"JSON Schema for source params."`
}

// Service provides source type metadata from the registry.
type Service struct {
	registry *sources.Registry
}

// NewService creates a new source types service.
func NewService(registry *sources.Registry) *Service {
	return &Service{registry: registry}
}

// ListSourceTypes returns metadata for all registered source types.
func (s *Service) ListSourceTypes() ([]SourceTypeMetadata, error) {
	if s.registry == nil {
		return nil, ErrRegistryUnavailable
	}

	srcs := s.registry.List()
	result := make([]SourceTypeMetadata, 0, len(srcs))
	for _, src := range srcs {
		result = append(result, toMetadata(src))
	}
	return result, nil
}

// GetSourceType returns metadata for a specific source type by name.
func (s *Service) GetSourceType(typeName string) (*SourceTypeMetadata, error) {
	if s.registry == nil {
		return nil, ErrRegistryUnavailable
	}

	src := s.registry.Get(typeName)
	if src == nil {
		return nil, ErrSourceTypeNotFound
	}
	metadata := toMetadata(src)
	return &metadata, nil
}

// toMetadata converts a sources.Source to SourceTypeMetadata.
func toMetadata(src sources.Source) SourceTypeMetadata {
	return SourceTypeMetadata{
		TypeName:           src.TypeName(),
		DisplayName:        src.DisplayName(),
		DefaultLookupCount: src.DefaultLookupCount(),
		ParamSchema:        src.ParamSchema(),
	}
}
