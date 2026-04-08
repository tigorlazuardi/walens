// Package reddit provides a template Reddit source implementation.
package reddit

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

// Source implements a template Reddit-based wallpaper source.
type Source struct{}

// Params defines configurable Reddit source options.
type Params struct {
	Target      string `json:"target" doc:"Reddit target to read from. Supports subreddit names, r/<name>, /r/<name>, /u/<name>, or /user/<name>."`
	Sort        string `json:"sort" doc:"Listing sort order: hot, new, top, rising."`
	TimeRange   string `json:"time_range" doc:"Time range used with top sort: hour, day, week, month, year, all."`
	SearchQuery string `json:"search_query" doc:"Optional Reddit search query to filter posts. Only applies to subreddit targets."`
	AllowNSFW   bool   `json:"allow_nsfw" doc:"Whether NSFW posts are allowed from this source."`
}

// TypeName implements sources.Source.
func (s *Source) TypeName() string {
	return "reddit"
}

// DisplayName implements sources.Source.
func (s *Source) DisplayName() string {
	return "Reddit"
}

// ValidateParams implements sources.Source.
func (s *Source) ValidateParams(raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("params are required")
	}

	var params Params
	if err := json.Unmarshal(raw, &params); err != nil {
		return fmt.Errorf("invalid params JSON: %w", err)
	}

	target, targetType, err := normalizeTarget(params.Target)
	if err != nil {
		return err
	}

	if targetType == targetTypeUser && params.SearchQuery != "" {
		return fmt.Errorf("search_query is not supported for user targets")
	}
	_ = target

	switch params.Sort {
	case "", "hot", "new", "top", "rising":
	default:
		return fmt.Errorf("invalid sort %q", params.Sort)
	}

	switch params.TimeRange {
	case "", "hour", "day", "week", "month", "year", "all":
	default:
		return fmt.Errorf("invalid time_range %q", params.TimeRange)
	}

	return nil
}

// ParamSchema implements sources.Source.
func (s *Source) ParamSchema() *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeObject,
		Description: "Configuration for a Reddit source.",
		Properties: map[string]*huma.Schema{
			"target": {
				Type:        huma.TypeString,
				Description: "Reddit target to read from. Supports subreddit names, r/<name>, /r/<name>, /u/<name>, or /user/<name>.",
			},
			"sort": {
				Type:        huma.TypeString,
				Description: "Listing sort order: hot, new, top, rising.",
				Enum:        []any{"hot", "new", "top", "rising"},
			},
			"time_range": {
				Type:        huma.TypeString,
				Description: "Time range used with top sort: hour, day, week, month, year, all.",
				Enum:        []any{"hour", "day", "week", "month", "year", "all"},
			},
			"search_query": {
				Type:        huma.TypeString,
				Description: "Optional Reddit search query to filter posts. Only applies to subreddit targets.",
			},
			"allow_nsfw": {
				Type:        huma.TypeBoolean,
				Description: "Whether NSFW posts are allowed from this source.",
			},
		},
		Required: []string{"target"},
	}
}

// DefaultLookupCount implements sources.Source.
func (s *Source) DefaultLookupCount() int {
	return 300
}

// Fetch implements sources.Source with a placeholder iterator.
func (s *Source) Fetch(ctx context.Context, req sources.FetchRequest) iter.Seq2[sources.ImageMetadata, error] {
	return func(yield func(sources.ImageMetadata, error) bool) {
		_ = ctx
		_ = req
	}
}

// BuildUniqueID implements sources.Source.
func (s *Source) BuildUniqueID(item sources.ImageMetadata) (string, error) {
	if item.SourceItemID == "" {
		return "", fmt.Errorf("source item ID is required for unique ID generation")
	}
	return fmt.Sprintf("reddit:%s", item.SourceItemID), nil
}

// BuildListingURL constructs a Reddit JSON listing URL for future fetch implementation.
func BuildListingURL(params Params, limit int) (string, error) {
	target, targetType, err := normalizeTarget(params.Target)
	if err != nil {
		return "", err
	}

	sortOrder := params.Sort
	if sortOrder == "" {
		sortOrder = "hot"
	}

	baseURL := &url.URL{
		Scheme: "https",
		Host:   "www.reddit.com",
		Path:   buildTargetPath(targetType, target, sortOrder),
	}

	q := baseURL.Query()
	q.Set("limit", fmt.Sprintf("%d", limit))
	if params.SearchQuery != "" {
		if targetType == targetTypeUser {
			return "", fmt.Errorf("search_query is not supported for user targets")
		}
		q.Set("q", params.SearchQuery)
		q.Set("restrict_sr", "1")
		baseURL.Path = fmt.Sprintf("/r/%s/search.json", target)
	}
	if params.TimeRange != "" {
		q.Set("t", params.TimeRange)
	}

	baseURL.RawQuery = q.Encode()
	return baseURL.String(), nil
}

type redditTargetType string

const (
	targetTypeSubreddit redditTargetType = "subreddit"
	targetTypeUser      redditTargetType = "user"
)

func normalizeTarget(raw string) (string, redditTargetType, error) {
	target := strings.TrimSpace(raw)
	target = strings.TrimPrefix(target, "https://www.reddit.com")
	target = strings.TrimPrefix(target, "https://reddit.com")
	target = strings.TrimPrefix(target, "http://www.reddit.com")
	target = strings.TrimPrefix(target, "http://reddit.com")
	target = strings.TrimSpace(target)

	if target == "" {
		return "", "", fmt.Errorf("target is required")
	}

	trimmed := strings.Trim(target, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) >= 2 {
		switch parts[0] {
		case "r":
			if parts[1] == "" {
				return "", "", fmt.Errorf("subreddit target is required")
			}
			return parts[1], targetTypeSubreddit, nil
		case "u", "user":
			if parts[1] == "" {
				return "", "", fmt.Errorf("user target is required")
			}
			return parts[1], targetTypeUser, nil
		}
	}

	if strings.HasPrefix(trimmed, "u/") || strings.HasPrefix(trimmed, "user/") || strings.HasPrefix(trimmed, "/u/") || strings.HasPrefix(trimmed, "/user/") {
		return "", "", fmt.Errorf("invalid user target %q", raw)
	}

	return strings.TrimPrefix(strings.TrimPrefix(trimmed, "r/"), "/r/"), targetTypeSubreddit, nil
}

func buildTargetPath(targetType redditTargetType, target string, sortOrder string) string {
	if targetType == targetTypeUser {
		return fmt.Sprintf("/user/%s/submitted/%s.json", target, sortOrder)
	}
	return fmt.Sprintf("/r/%s/%s.json", target, sortOrder)
}

// New returns a Reddit source implementation instance.
func New() *Source {
	return &Source{}
}
