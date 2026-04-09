// Package booru provides a simple tag-based image board source implementation.
package booru

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/sources"
)

var booruHTTPClient = http.DefaultClient

// BooruSource implements a simple tag-based image board source.
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

// booruPostsResponse represents the XML response from the Gelbooru API.
type booruPostsResponse struct {
	Count  int         `xml:"count,attr"`
	Offset int         `xml:"offset,attr"`
	Posts  []booruPost `xml:"post"`
}

// booruPost represents a single post element in the Gelbooru XML response.
type booruPost struct {
	XMLName    xml.Name `xml:"post"`
	ID         string   `xml:"id,attr"`
	PreviewURL string   `xml:"preview_url,attr"`
	FileURL    string   `xml:"file_url,attr"`
	Tags       string   `xml:"tags,attr"`
	Rating     string   `xml:"rating,attr"`
	Score      int      `xml:"score,attr"`
	Width      int      `xml:"width,attr"`
	Height     int      `xml:"height,attr"`
	Source     string   `xml:"source,attr"`
	CreatorID  string   `xml:"creator_id,attr"`
	MD5        string   `xml:"md5,attr"`
}

// Fetch implements sources.Source by querying the booru API.
func (s *BooruSource) Fetch(ctx context.Context, req sources.FetchRequest) iter.Seq2[sources.ImageMetadata, error] {
	return func(yield func(sources.ImageMetadata, error) bool) {
		if err := ctx.Err(); err != nil {
			yield(sources.ImageMetadata{}, err)
			return
		}

		var params BooruParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			yield(sources.ImageMetadata{}, fmt.Errorf("invalid params JSON: %w", err))
			return
		}

		remaining := req.LookupCount
		if remaining <= 0 {
			remaining = s.DefaultLookupCount()
		}

		pageOffset := 0
		pageLimit := 100
		if pageLimit > remaining {
			pageLimit = remaining
		}

		for remaining > 0 {
			if err := ctx.Err(); err != nil {
				yield(sources.ImageMetadata{}, err)
				return
			}

			apiURL, err := buildBooruURL(params.BooruHost, params, pageLimit, pageOffset)
			if err != nil {
				yield(sources.ImageMetadata{}, fmt.Errorf("build booru URL: %w", err))
				return
			}

			posts, err := fetchPosts(ctx, apiURL)
			if err != nil {
				yield(sources.ImageMetadata{}, fmt.Errorf("fetch posts: %w", err))
				return
			}

			if len(posts) == 0 {
				return
			}

			for _, post := range posts {
				if err := ctx.Err(); err != nil {
					yield(sources.ImageMetadata{}, err)
					return
				}

				remaining--
				metadata := postToImageMetadata(post)
				if metadata.SourceItemID == "" {
					continue
				}
				if !yield(metadata, nil) {
					return
				}
				if remaining == 0 {
					return
				}
			}

			pageOffset++
		}
	}
}

// fetchPosts retrieves and parses posts from the booru API.
func fetchPosts(ctx context.Context, apiURL string) ([]booruPost, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "walens/0.1 (+https://github.com/walens/walens)")

	resp, err := booruHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch posts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("booru request failed with status %d", resp.StatusCode)
	}

	var postsResp booruPostsResponse
	if err := xml.NewDecoder(resp.Body).Decode(&postsResp); err != nil {
		return nil, fmt.Errorf("decode XML response: %w", err)
	}

	return postsResp.Posts, nil
}

// postToImageMetadata transforms a booru post into ImageMetadata.
func postToImageMetadata(post booruPost) sources.ImageMetadata {
	if post.ID == "" || post.FileURL == "" {
		return sources.ImageMetadata{}
	}

	aspectRatio := 0.0
	if post.Width > 0 && post.Height > 0 {
		aspectRatio = float64(post.Width) / float64(post.Height)
	}

	tags := strings.Fields(post.Tags)

	isAdult := false
	switch strings.ToLower(post.Rating) {
	case "explicit", "questionable":
		isAdult = true
	}

	mimeType := detectImageMimeType(post.FileURL)

	return sources.ImageMetadata{
		UniqueIdentifier: post.MD5,
		PreviewURL:       post.PreviewURL,
		OriginURL:        post.FileURL,
		SourceItemID:     post.ID,
		OriginalID:       post.ID,
		Uploader:         post.CreatorID,
		MimeType:         mimeType,
		Width:            post.Width,
		Height:           post.Height,
		AspectRatio:      aspectRatio,
		IsAdult:          isAdult,
		Tags:             tags,
	}
}

// detectImageMimeType determines the MIME type from a file URL extension.
func detectImageMimeType(fileURL string) string {
	parsed, err := url.Parse(fileURL)
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

// BuildUniqueID implements sources.Source.
func (s *BooruSource) BuildUniqueID(item sources.ImageMetadata) (string, error) {
	if item.SourceItemID == "" {
		return "", fmt.Errorf("source item ID is required for unique ID generation")
	}
	return fmt.Sprintf("booru:%s", item.SourceItemID), nil
}

// buildBooruURL constructs the API URL for a booru query.
func buildBooruURL(host string, params BooruParams, limit int, pid int) (string, error) {
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
	q.Set("pid", fmt.Sprintf("%d", pid))

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
