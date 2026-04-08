// Package reddit provides a template Reddit source implementation.
package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"iter"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/sources"
)

const redditPageLimit = 100

var (
	redditHTTPClient = http.DefaultClient
	redditBaseURL    = "https://www.reddit.com"
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

// Fetch implements sources.Source with paginated Reddit JSON listing requests.
func (s *Source) Fetch(ctx context.Context, req sources.FetchRequest) iter.Seq2[sources.ImageMetadata, error] {
	return func(yield func(sources.ImageMetadata, error) bool) {
		if err := ctx.Err(); err != nil {
			yield(sources.ImageMetadata{}, err)
			return
		}

		var params Params
		if err := json.Unmarshal(req.Params, &params); err != nil {
			yield(sources.ImageMetadata{}, fmt.Errorf("invalid params JSON: %w", err))
			return
		}
		if err := s.ValidateParams(req.Params); err != nil {
			yield(sources.ImageMetadata{}, err)
			return
		}

		remaining := req.LookupCount
		if remaining <= 0 {
			remaining = s.DefaultLookupCount()
		}

		after := ""
		for remaining > 0 {
			pageSize := remaining
			if pageSize > redditPageLimit {
				pageSize = redditPageLimit
			}

			listingURL, err := BuildListingURL(params, pageSize, after)
			if err != nil {
				yield(sources.ImageMetadata{}, err)
				return
			}

			listing, err := fetchListing(ctx, listingURL)
			if err != nil {
				yield(sources.ImageMetadata{}, err)
				return
			}

			if len(listing.Data.Children) == 0 {
				return
			}

			for _, child := range listing.Data.Children {
				if err := ctx.Err(); err != nil {
					yield(sources.ImageMetadata{}, err)
					return
				}

				remaining--
				if child.Data.IsGallery {
					for _, metadata := range listingChildToAlbumMetadata(child.Data, params.AllowNSFW) {
						if !yield(metadata, nil) {
							return
						}
					}
					if remaining == 0 {
						return
					}
					continue
				}

				if metadata, ok := listingChildToMetadata(child.Data, params.AllowNSFW); ok {
					if !yield(metadata, nil) {
						return
					}
				}

				if remaining == 0 {
					return
				}
			}

			after = listing.Data.After
			if after == "" {
				return
			}
		}
	}
}

// BuildUniqueID implements sources.Source.
func (s *Source) BuildUniqueID(item sources.ImageMetadata) (string, error) {
	if item.SourceItemID == "" {
		return "", fmt.Errorf("source item ID is required for unique ID generation")
	}
	return fmt.Sprintf("reddit:%s", item.SourceItemID), nil
}

// BuildListingURL constructs a Reddit JSON listing URL.
func BuildListingURL(params Params, limit int, after string) (string, error) {
	target, targetType, err := normalizeTarget(params.Target)
	if err != nil {
		return "", err
	}

	sortOrder := params.Sort
	if sortOrder == "" {
		sortOrder = "hot"
	}

	baseURL, err := url.Parse(redditBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse reddit base URL: %w", err)
	}
	baseURL.Path = path.Join(baseURL.Path, buildTargetPath(targetType, target, sortOrder))

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
	if after != "" {
		q.Set("after", after)
	}

	baseURL.RawQuery = q.Encode()
	return baseURL.String(), nil
}

type redditListing struct {
	Data struct {
		After    string               `json:"after"`
		Children []redditListingChild `json:"children"`
	} `json:"data"`
}

type redditListingChild struct {
	Data redditPost `json:"data"`
}

type redditPost struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Author      string `json:"author"`
	Title       string `json:"title"`
	Permalink   string `json:"permalink"`
	URL         string `json:"url"`
	Thumbnail   string `json:"thumbnail"`
	PostHint    string `json:"post_hint"`
	IsSelf      bool   `json:"is_self"`
	IsVideo     bool   `json:"is_video"`
	Media       any    `json:"media"`
	Over18      bool   `json:"over_18"`
	IsGallery   bool   `json:"is_gallery"`
	GalleryData struct {
		Items []struct {
			MediaID string `json:"media_id"`
		} `json:"items"`
	} `json:"gallery_data"`
	MediaMetadata map[string]struct {
		Status string `json:"status"`
		Mime   string `json:"m"`
		S      struct {
			U string `json:"u"`
			X int    `json:"x"`
			Y int    `json:"y"`
		} `json:"s"`
	} `json:"media_metadata"`
	Preview struct {
		Images []struct {
			Source struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"source"`
		} `json:"images"`
	} `json:"preview"`
}

func fetchListing(ctx context.Context, listingURL string) (*redditListing, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "walens/0.1 (+https://github.com/tigorlazuardi/walens)")

	resp, err := redditHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch reddit listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("reddit listing request failed with status %d", resp.StatusCode)
	}

	var listing redditListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, fmt.Errorf("decode reddit listing: %w", err)
	}
	return &listing, nil
}

func listingChildToMetadata(post redditPost, allowNSFW bool) (sources.ImageMetadata, bool) {
	if post.Over18 && !allowNSFW {
		return sources.ImageMetadata{}, false
	}

	originURL := decodeRedditURL(post.URL)
	previewURL := decodeRedditURL(post.Thumbnail)
	width := 0
	height := 0

	if len(post.Preview.Images) > 0 {
		source := post.Preview.Images[0].Source
		if source.URL != "" {
			originURL = decodeRedditURL(source.URL)
			width = source.Width
			height = source.Height
		}
		if previewURL == "" {
			previewURL = decodeRedditURL(source.URL)
		}
	}

	if !isLikelyImagePost(post, originURL) {
		return sources.ImageMetadata{}, false
	}

	mimeType := detectImageMimeType(originURL)
	if mimeType == "" {
		return sources.ImageMetadata{}, false
	}

	itemID := firstNonEmpty(post.Name, post.ID)
	if itemID == "" {
		return sources.ImageMetadata{}, false
	}

	aspectRatio := 0.0
	if width > 0 && height > 0 {
		aspectRatio = float64(width) / float64(height)
	}

	tags := make([]string, 0, 1)
	if post.Title != "" {
		tags = append(tags, post.Title)
	}

	return sources.ImageMetadata{
		PreviewURL:   previewURL,
		OriginURL:    originURL,
		SourceItemID: itemID,
		OriginalID:   post.ID,
		Uploader:     post.Author,
		MimeType:     mimeType,
		Width:        width,
		Height:       height,
		AspectRatio:  aspectRatio,
		IsAdult:      post.Over18,
		Tags:         tags,
	}, true
}

func listingChildToAlbumMetadata(post redditPost, allowNSFW bool) []sources.ImageMetadata {
	if post.Over18 && !allowNSFW {
		return nil
	}

	results := make([]sources.ImageMetadata, 0, len(post.GalleryData.Items))
	parentID := firstNonEmpty(post.Name, post.ID)
	for _, galleryItem := range post.GalleryData.Items {
		media, ok := post.MediaMetadata[galleryItem.MediaID]
		if !ok || media.Status != "valid" {
			continue
		}

		originURL := decodeRedditURL(media.S.U)
		mimeType := media.Mime
		if mimeType == "" {
			mimeType = detectImageMimeType(originURL)
		}
		if !strings.HasPrefix(mimeType, "image/") {
			continue
		}

		aspectRatio := 0.0
		if media.S.X > 0 && media.S.Y > 0 {
			aspectRatio = float64(media.S.X) / float64(media.S.Y)
		}

		results = append(results, sources.ImageMetadata{
			PreviewURL:   originURL,
			OriginURL:    originURL,
			SourceItemID: parentID + ":" + galleryItem.MediaID,
			OriginalID:   galleryItem.MediaID,
			Uploader:     post.Author,
			MimeType:     mimeType,
			Width:        media.S.X,
			Height:       media.S.Y,
			AspectRatio:  aspectRatio,
			IsAdult:      post.Over18,
			Tags:         []string{post.Title},
		})
	}

	return results
}

func decodeRedditURL(raw string) string {
	decoded := html.UnescapeString(strings.TrimSpace(raw))
	if decoded == "" || decoded == "self" || decoded == "default" || decoded == "nsfw" || decoded == "spoiler" {
		return ""
	}
	return decoded
}

func isLikelyImagePost(post redditPost, rawURL string) bool {
	if post.IsGallery {
		return false
	}
	if post.IsSelf || post.IsVideo {
		return false
	}
	switch post.PostHint {
	case "hosted:video", "rich:video", "link":
		// handled below only when URL itself is a direct image
	case "self":
		return false
	}
	if post.PostHint == "image" {
		return true
	}
	return detectImageMimeType(rawURL) != ""
}

func detectImageMimeType(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(parsed.Path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
