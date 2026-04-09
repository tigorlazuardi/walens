package booru

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/walens/walens/internal/sources"
)

func TestBuildBooruURL(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		params BooruParams
		limit  int
		pid    int
		want   []string
	}{
		{
			name: "basic URL with tags",
			host: "gelbooru.com",
			params: BooruParams{
				Tags: []string{"landscape", "nature"},
			},
			limit: 50,
			pid:   0,
			want:  []string{"page=dapi", "s=post", "q=index", "limit=50", "pid=0", "tags=landscape%2Bnature"},
		},
		{
			name: "URL with rating filter",
			host: "gelbooru.com",
			params: BooruParams{
				Tags:   []string{"anime"},
				Rating: "safe",
			},
			limit: 25,
			pid:   1,
			want:  []string{"rating=safe", "tags=anime", "limit=25", "pid=1"},
		},
		{
			name: "URL with custom host",
			host: "custom.booru.org",
			params: BooruParams{
				Tags: []string{"wallpaper"},
			},
			limit: 100,
			pid:   0,
			want:  []string{"custom.booru.org", "tags=wallpaper"},
		},
		{
			name: "URL with all params",
			host: "gelbooru.com",
			params: BooruParams{
				Tags:     []string{"art", "digital"},
				Rating:   "questionable",
				MinScore: 10,
				ApiKey:   "my-api-key",
				UserID:   "my-user-id",
			},
			limit: 50,
			pid:   2,
			want:  []string{"rating=questionable", "min_score=10", "api_key=my-api-key", "user_id=my-user-id", "tags=art%2Bdigital"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildBooruURL(tt.host, tt.params, tt.limit, tt.pid)
			if err != nil {
				t.Fatalf("buildBooruURL() error = %v", err)
			}
			for _, wantPart := range tt.want {
				if !strings.Contains(got, wantPart) {
					t.Errorf("buildBooruURL() = %q, want to contain %q", got, wantPart)
				}
			}
		})
	}
}

func TestBuildBooruURLDefaultHost(t *testing.T) {
	url, err := buildBooruURL("", BooruParams{Tags: []string{"test"}}, 50, 0)
	if err != nil {
		t.Fatalf("buildBooruURL() error = %v", err)
	}
	if !strings.Contains(url, "gelbooru.com") {
		t.Errorf("buildBooruURL() default host = %q, want to contain gelbooru.com", url)
	}
}

func TestBuildUniqueID(t *testing.T) {
	src := New()
	tests := []struct {
		name    string
		item    sources.ImageMetadata
		want    string
		wantErr bool
	}{
		{
			name:    "valid source item ID",
			item:    sources.ImageMetadata{SourceItemID: "12345"},
			want:    "booru:12345",
			wantErr: false,
		},
		{
			name:    "empty source item ID",
			item:    sources.ImageMetadata{SourceItemID: ""},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := src.BuildUniqueID(tt.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildUniqueID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("BuildUniqueID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostToImageMetadata(t *testing.T) {
	tests := []struct {
		name string
		post booruPost
		want sources.ImageMetadata
	}{
		{
			name: "basic post",
			post: booruPost{
				ID:         "12345",
				MD5:        "abc123",
				FileURL:    "https://example.com/image.jpg",
				PreviewURL: "https://example.com/thumb.jpg",
				Tags:       "landscape nature mountain",
				Rating:     "safe",
				Width:      1920,
				Height:     1080,
				Score:      100,
			},
			want: sources.ImageMetadata{
				UniqueIdentifier: "abc123",
				OriginURL:        "https://example.com/image.jpg",
				PreviewURL:       "https://example.com/thumb.jpg",
				SourceItemID:     "12345",
				OriginalID:       "12345",
				MimeType:         "image/jpeg",
				Width:            1920,
				Height:           1080,
				AspectRatio:      1.7777777777777777,
				IsAdult:          false,
				Tags:             []string{"landscape", "nature", "mountain"},
			},
		},
		{
			name: "adult post",
			post: booruPost{
				ID:      "67890",
				MD5:     "def456",
				FileURL: "https://example.com/explicit.png",
				Tags:    "hentai",
				Rating:  "explicit",
				Width:   1000,
				Height:  1500,
			},
			want: sources.ImageMetadata{
				UniqueIdentifier: "def456",
				OriginURL:        "https://example.com/explicit.png",
				SourceItemID:     "67890",
				OriginalID:       "67890",
				MimeType:         "image/png",
				Width:            1000,
				Height:           1500,
				AspectRatio:      0.6666666666666666,
				IsAdult:          true,
				Tags:             []string{"hentai"},
			},
		},
		{
			name: "questionable rating is adult",
			post: booruPost{
				ID:      "11111",
				MD5:     "ghi789",
				FileURL: "https://example.com/questionable.webp",
				Rating:  "questionable",
			},
			want: sources.ImageMetadata{
				UniqueIdentifier: "ghi789",
				OriginURL:        "https://example.com/questionable.webp",
				SourceItemID:     "11111",
				OriginalID:       "11111",
				MimeType:         "image/webp",
				IsAdult:          true,
			},
		},
		{
			name: "empty ID is filtered",
			post: booruPost{
				ID:      "",
				FileURL: "https://example.com/image.jpg",
			},
			want: sources.ImageMetadata{
				SourceItemID: "",
			},
		},
		{
			name: "empty file URL is filtered",
			post: booruPost{
				ID:      "123",
				FileURL: "",
			},
			want: sources.ImageMetadata{
				SourceItemID: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postToImageMetadata(tt.post)
			if got.SourceItemID != tt.want.SourceItemID {
				t.Errorf("SourceItemID = %v, want %v", got.SourceItemID, tt.want.SourceItemID)
			}
			if got.UniqueIdentifier != tt.want.UniqueIdentifier {
				t.Errorf("UniqueIdentifier = %v, want %v", got.UniqueIdentifier, tt.want.UniqueIdentifier)
			}
			if got.OriginURL != tt.want.OriginURL {
				t.Errorf("OriginURL = %v, want %v", got.OriginURL, tt.want.OriginURL)
			}
			if got.IsAdult != tt.want.IsAdult {
				t.Errorf("IsAdult = %v, want %v", got.IsAdult, tt.want.IsAdult)
			}
			if got.Width != tt.want.Width || got.Height != tt.want.Height {
				t.Errorf("Width/Height = %v/%v, want %v/%v", got.Width, got.Height, tt.want.Width, tt.want.Height)
			}
			if got.AspectRatio != tt.want.AspectRatio {
				t.Errorf("AspectRatio = %v, want %v", got.AspectRatio, tt.want.AspectRatio)
			}
			if got.MimeType != tt.want.MimeType {
				t.Errorf("MimeType = %v, want %v", got.MimeType, tt.want.MimeType)
			}
			if len(got.Tags) != len(tt.want.Tags) {
				t.Errorf("Tags len = %v, want %v", len(got.Tags), len(tt.want.Tags))
			}
		})
	}
}

func TestDetectImageMimeType(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"JPEG", "https://example.com/image.jpg", "image/jpeg"},
		{"JPEG uppercase", "https://example.com/image.JPEG", "image/jpeg"},
		{"PNG", "https://example.com/image.png", "image/png"},
		{"WebP", "https://example.com/image.webp", "image/webp"},
		{"GIF", "https://example.com/image.gif", "image/gif"},
		{"Unknown", "https://example.com/image.bmp", ""},
		{"No extension", "https://example.com/image", ""},
		{"Invalid URL", "://invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectImageMimeType(tt.url)
			if got != tt.want {
				t.Errorf("detectImageMimeType(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestFetchRespectsLookupBudget(t *testing.T) {
	oldClient := booruHTTPClient
	defer func() { booruHTTPClient = oldClient }()

	requestCount := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, postsXML(10, requestCount))
	}))
	defer server.Close()
	booruHTTPClient = server.Client()

	src := New()
	params, _ := json.Marshal(BooruParams{
		Tags:      []string{"test"},
		BooruHost: strings.TrimPrefix(server.URL, "https://"),
	})

	seq := src.Fetch(context.Background(), sources.FetchRequest{Params: params, LookupCount: 25})

	var yielded []sources.ImageMetadata
	for item, err := range seq {
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}
		yielded = append(yielded, item)
	}

	if requestCount != 3 {
		t.Fatalf("expected 3 requests (10 + 10 + 5), got %d", requestCount)
	}
	if len(yielded) != 25 {
		t.Fatalf("expected 25 yielded items, got %d", len(yielded))
	}
}

func TestFetchStopsWhenConsumerStops(t *testing.T) {
	oldClient := booruHTTPClient
	defer func() { booruHTTPClient = oldClient }()

	requestCount := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, postsXML(100, requestCount))
	}))
	defer server.Close()
	booruHTTPClient = server.Client()

	src := New()
	params, _ := json.Marshal(BooruParams{
		Tags:      []string{"test"},
		BooruHost: strings.TrimPrefix(server.URL, "https://"),
	})

	seq := src.Fetch(context.Background(), sources.FetchRequest{Params: params, LookupCount: 100})

	count := 0
	for item, err := range seq {
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}
		_ = item
		count++
		if count == 5 {
			break
		}
	}

	if requestCount != 1 {
		t.Fatalf("expected 1 request, got %d", requestCount)
	}
}

func TestFetchHandlesContextCancellation(t *testing.T) {
	oldClient := booruHTTPClient
	defer func() { booruHTTPClient = oldClient }()

	requestCount := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, postsXML(100, requestCount))
	}))
	defer server.Close()
	booruHTTPClient = server.Client()

	src := New()
	params, _ := json.Marshal(BooruParams{
		Tags:      []string{"test"},
		BooruHost: strings.TrimPrefix(server.URL, "https://"),
	})

	ctx, cancel := context.WithCancel(context.Background())
	seq := src.Fetch(ctx, sources.FetchRequest{Params: params, LookupCount: 100})

	count := 0
	for item, err := range seq {
		if err != nil {
			if err == context.Canceled {
				return
			}
			t.Fatalf("unexpected error = %v", err)
		}
		_ = item
		count++
		if count == 3 {
			cancel()
		}
	}

	if count != 3 {
		t.Fatalf("expected iteration to stop at 3 items after cancellation")
	}
}

func TestFetchEmptyResponse(t *testing.T) {
	oldClient := booruHTTPClient
	defer func() { booruHTTPClient = oldClient }()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><posts count="0" offset="0"></posts>`)
	}))
	defer server.Close()
	booruHTTPClient = server.Client()

	src := New()
	params, _ := json.Marshal(BooruParams{
		Tags:      []string{"nonexistent"},
		BooruHost: strings.TrimPrefix(server.URL, "https://"),
	})

	seq := src.Fetch(context.Background(), sources.FetchRequest{Params: params, LookupCount: 100})

	count := 0
	for item, err := range seq {
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}
		_ = item
		count++
	}

	if count != 0 {
		t.Fatalf("expected 0 items, got %d", count)
	}
}

func TestXMLParsing(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<posts count="2" offset="0">
	<post id="1" file_url="https://example.com/1.jpg" preview_url="https://example.com/1_thumb.jpg" tags="tag1 tag2" rating="safe" score="10" width="100" height="200" md5="abc123" creator_id="user1" source=""/>
	<post id="2" file_url="https://example.com/2.png" preview_url="https://example.com/2_thumb.png" tags="tag3" rating="explicit" score="20" width="300" height="400" md5="def456" creator_id="user2" source=""/>
</posts>`

	var resp booruPostsResponse
	if err := xml.Unmarshal([]byte(xmlData), &resp); err != nil {
		t.Fatalf("failed to parse XML: %v", err)
	}

	if resp.Count != 2 {
		t.Errorf("Count = %d, want 2", resp.Count)
	}
	if len(resp.Posts) != 2 {
		t.Errorf("Posts len = %d, want 2", len(resp.Posts))
	}

	if resp.Posts[0].ID != "1" {
		t.Errorf("Posts[0].ID = %s, want 1", resp.Posts[0].ID)
	}
	if resp.Posts[0].Rating != "safe" {
		t.Errorf("Posts[0].Rating = %s, want safe", resp.Posts[0].Rating)
	}
	if resp.Posts[1].Rating != "explicit" {
		t.Errorf("Posts[1].Rating = %s, want explicit", resp.Posts[1].Rating)
	}
}

func postsXML(count int, page int) string {
	posts := make([]string, count)
	for i := 0; i < count; i++ {
		id := (page-1)*count + i + 1
		posts[i] = fmt.Sprintf(`<post id="%d" file_url="https://example.com/%d.jpg" preview_url="https://example.com/%d_thumb.jpg" tags="tag1 tag2" rating="safe" score="10" width="100" height="200" md5="hash%d" creator_id="user1"/>`, id, id, id, id)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><posts count="%d" offset="%d">%s</posts>`, count, page-1, strings.Join(posts, ""))
}
