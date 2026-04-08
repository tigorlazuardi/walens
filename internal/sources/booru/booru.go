// Package booru provides a simple tag-based image board source implementation.
package booru

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/sources"
)

// BooruSource implements a simple tag-based image board source.
// This is a placeholder implementation for demonstration purposes.
type BooruSource struct{}

// BooruParams defines the configuration for a booru source.
type BooruParams struct {
	Tags      []string `json:"tags" doc:"List of tags to filter images by."`
	Rating    string   `json:"rating" doc:"Content rating filter: safe, questionable, explicit."`
	MinScore  int      `json:"min_score" doc:"Minimum image score."`
	BooruHost string   `json:"booru_host" doc:"The booru instance host (e.g., gelbooru.com)."`
	ApiKey    string   `json:"api_key" doc:"Optional API key for authenticated requests."`
	UserID    string   `json:"user_id" doc:"Optional user ID for authenticated requests."`
}

// TypeName implements sources.Source.
func (s *BooruSource) TypeName() string {
	return "booru"
}

// DisplayName implements sources.Source.
func (s *BooruSource) DisplayName() string {
	return "Booru Image Board"
}

// ValidateParams implements sources.Source.
func (s *BooruSource) ValidateParams(raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("params are required")
	}
	var params BooruParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return fmt.Errorf("invalid params JSON: %w", err)
	}
	if len(params.Tags) == 0 && params.BooruHost == "" {
		return fmt.Errorf("at least one tag or booru_host is required")
	}
	return nil
}

// ParamSchema implements sources.Source.
func (s *BooruSource) ParamSchema() *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeObject,
		Description: "Configuration for a booru image board source.",
		Properties: map[string]*huma.Schema{
			"tags": {
				Type:        huma.TypeArray,
				Description: "List of tags to filter images by.",
				Items: &huma.Schema{
					Type:   huma.TypeString,
					Format: "string",
				},
			},
			"rating": {
				Type:        huma.TypeString,
				Description: "Content rating filter: safe, questionable, explicit.",
				Enum:        []any{"safe", "questionable", "explicit"},
			},
			"min_score": {
				Type:        huma.TypeInteger,
				Description: "Minimum image score.",
				Minimum:     ptr[float64](0),
			},
			"booru_host": {
				Type:        huma.TypeString,
				Description: "The booru instance host (e.g., gelbooru.com).",
				Format:      "string",
			},
			"api_key": {
				Type:        huma.TypeString,
				Description: "Optional API key for authenticated requests.",
				Format:      "string",
			},
			"user_id": {
				Type:        huma.TypeString,
				Description: "Optional user ID for authenticated requests.",
				Format:      "string",
			},
		},
		Required: []string{"tags"},
	}
}

// DefaultLookupCount implements sources.Source.
func (s *BooruSource) DefaultLookupCount() int {
	return 100
}

// Fetch implements sources.Source by returning an empty iterator.
// This is a placeholder - real implementation would query the booru API.
func (s *BooruSource) Fetch(ctx context.Context, req sources.FetchRequest) iter.Seq2[sources.ImageMetadata, error] {
	return func(yield func(sources.ImageMetadata, error) bool) {
		// Placeholder: yield no images
		// Real implementation would:
		// 1. Parse params
		// 2. Build API request to booru host
		// 3. Paginate through results
		// 4. Yield ImageMetadata items
		_ = ctx
		_ = req
	}
}

// BuildUniqueID implements sources.Source.
func (s *BooruSource) BuildUniqueID(item sources.ImageMetadata) (string, error) {
	if item.SourceItemID == "" {
		return "", fmt.Errorf("source item ID is required for unique ID generation")
	}
	return fmt.Sprintf("booru:%s", item.SourceItemID), nil
}

// buildBooruURL constructs the API URL for a booru query.
func buildBooruURL(host string, params BooruParams, limit int) (string, error) {
	if host == "" {
		host = "gelbooru.com"
	}

	baseURL := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/index.php",
	}

	q := baseURL.Query()
	q.Set("page", "dapi")
	q.Set("s", "post")
	q.Set("q", "index")
	q.Set("limit", fmt.Sprintf("%d", limit))

	if len(params.Tags) > 0 {
		q.Set("tags", strings.Join(params.Tags, "+"))
	}
	if params.Rating != "" {
		q.Set("rating", params.Rating)
	}
	if params.MinScore > 0 {
		q.Set("min_score", fmt.Sprintf("%d", params.MinScore))
	}
	if params.ApiKey != "" {
		q.Set("api_key", params.ApiKey)
	}
	if params.UserID != "" {
		q.Set("user_id", params.UserID)
	}

	baseURL.RawQuery = q.Encode()
	return baseURL.String(), nil
}

// ptr returns a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}

// New returns a booru source implementation instance.
func New() *BooruSource {
	return &BooruSource{}
}
