package reddit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/walens/walens/internal/sources"
)

func TestSourceValidateParams(t *testing.T) {
	src := New()

	valid, _ := json.Marshal(Params{Target: "wallpapers", Sort: "top", TimeRange: "week"})
	if err := src.ValidateParams(valid); err != nil {
		t.Fatalf("ValidateParams(valid) error = %v", err)
	}

	validUser, _ := json.Marshal(Params{Target: "/u/spez", Sort: "new"})
	if err := src.ValidateParams(validUser); err != nil {
		t.Fatalf("ValidateParams(valid user target) error = %v", err)
	}

	invalid, _ := json.Marshal(Params{Target: "", Sort: "weird"})
	if err := src.ValidateParams(invalid); err == nil {
		t.Fatal("ValidateParams(invalid) expected error")
	}

	invalidUserSearch, _ := json.Marshal(Params{Target: "/user/spez", SearchQuery: "foo"})
	if err := src.ValidateParams(invalidUserSearch); err == nil {
		t.Fatal("ValidateParams(invalid user search) expected error")
	}
}

func TestBuildListingURL(t *testing.T) {
	url, err := BuildListingURL(Params{
		Target:      "wallpapers",
		Sort:        "top",
		TimeRange:   "month",
		SearchQuery: "landscape",
	}, 25, "")
	if err != nil {
		t.Fatalf("BuildListingURL() error = %v", err)
	}

	wantParts := []string{"/r/wallpapers/search.json", "limit=25", "q=landscape", "restrict_sr=1", "t=month"}
	for _, want := range wantParts {
		if !strings.Contains(url, want) {
			t.Fatalf("BuildListingURL() = %q, want substring %q", url, want)
		}
	}
}

func TestBuildListingURLUserTarget(t *testing.T) {
	url, err := BuildListingURL(Params{
		Target:    "/u/spez",
		Sort:      "new",
		TimeRange: "week",
	}, 10, "")
	if err != nil {
		t.Fatalf("BuildListingURL(user) error = %v", err)
	}

	wantParts := []string{"/user/spez/submitted/new.json", "limit=10", "t=week"}
	for _, want := range wantParts {
		if !strings.Contains(url, want) {
			t.Fatalf("BuildListingURL(user) = %q, want substring %q", url, want)
		}
	}
}

func TestBuildListingURLWithAfter(t *testing.T) {
	url, err := BuildListingURL(Params{Target: "wallpapers"}, 100, "t3_after")
	if err != nil {
		t.Fatalf("BuildListingURL(after) error = %v", err)
	}
	if !strings.Contains(url, "after=t3_after") {
		t.Fatalf("BuildListingURL(after) = %q, want after param", url)
	}
}

func TestNormalizeTarget(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantTarget string
		wantType   redditTargetType
		wantErr    bool
	}{
		{name: "plain subreddit", raw: "wallpapers", wantTarget: "wallpapers", wantType: targetTypeSubreddit},
		{name: "prefixed subreddit", raw: "/r/wallpapers", wantTarget: "wallpapers", wantType: targetTypeSubreddit},
		{name: "u prefix", raw: "/u/spez", wantTarget: "spez", wantType: targetTypeUser},
		{name: "user prefix", raw: "/user/spez", wantTarget: "spez", wantType: targetTypeUser},
		{name: "empty", raw: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, gotType, err := normalizeTarget(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotTarget != tt.wantTarget || gotType != tt.wantType {
				t.Fatalf("normalizeTarget() = (%q, %q), want (%q, %q)", gotTarget, gotType, tt.wantTarget, tt.wantType)
			}
		})
	}
}

func TestFetchRespectsLookupBudgetAcrossPages(t *testing.T) {
	oldBaseURL := redditBaseURL
	oldClient := redditHTTPClient
	defer func() {
		redditBaseURL = oldBaseURL
		redditHTTPClient = oldClient
	}()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, listingJSON(r.URL.Query().Get("after")))
	}))
	defer server.Close()

	redditBaseURL = server.URL
	redditHTTPClient = server.Client()

	params, _ := json.Marshal(Params{Target: "wallpapers"})
	seq := New().Fetch(context.Background(), sources.FetchRequest{Params: params, LookupCount: 150})

	var yielded []sources.ImageMetadata
	for item, err := range seq {
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}
		yielded = append(yielded, item)
	}

	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
	if len(yielded) != 149 {
		// 150 looked-up posts minus 2 non-image posts plus 1 extra image from a gallery post.
		t.Fatalf("expected 149 yielded image posts, got %d", len(yielded))
	}
	if yielded[0].SourceItemID == "" || yielded[len(yielded)-1].SourceItemID == "" {
		t.Fatal("expected yielded items to have source IDs")
	}
}

func TestFetchStopsWhenConsumerStops(t *testing.T) {
	oldBaseURL := redditBaseURL
	oldClient := redditHTTPClient
	defer func() {
		redditBaseURL = oldBaseURL
		redditHTTPClient = oldClient
	}()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, listingJSON(""))
	}))
	defer server.Close()

	redditBaseURL = server.URL
	redditHTTPClient = server.Client()

	params, _ := json.Marshal(Params{Target: "wallpapers"})
	seq := New().Fetch(context.Background(), sources.FetchRequest{Params: params, LookupCount: 120})

	count := 0
	for item, err := range seq {
		if err != nil {
			t.Fatalf("Fetch() error = %v", err)
		}
		_ = item
		count++
		if count == 1 {
			break
		}
	}

	if requestCount != 1 {
		t.Fatalf("expected 1 request, got %d", requestCount)
	}
}

func TestListingChildToMetadataFiltersNonImagePostKinds(t *testing.T) {
	tests := []struct {
		name string
		post redditPost
	}{
		{
			name: "self post",
			post: redditPost{ID: "1", Name: "t3_1", IsSelf: true, URL: "https://reddit.com/r/test/comments/1", PostHint: "self"},
		},
		{
			name: "hosted video",
			post: redditPost{ID: "2", Name: "t3_2", IsVideo: true, URL: "https://v.redd.it/abc", PostHint: "hosted:video"},
		},
		{
			name: "rich video",
			post: redditPost{ID: "3", Name: "t3_3", URL: "https://youtube.com/watch?v=abc", PostHint: "rich:video"},
		},
		{
			name: "comment thread link",
			post: redditPost{ID: "4", Name: "t3_4", URL: "https://reddit.com/r/test/comments/4/example", PostHint: "link"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := listingChildToMetadata(tt.post, true); ok {
				t.Fatal("expected post to be filtered out")
			}
		})
	}
}

func TestListingChildToAlbumMetadataSkipsNonImageMedia(t *testing.T) {
	post := redditPost{ID: "10", Name: "t3_10", Title: "gallery", Author: "user", IsGallery: true}
	post.GalleryData.Items = []struct {
		MediaID string `json:"media_id"`
	}{
		{MediaID: "img"},
		{MediaID: "vid"},
	}
	post.MediaMetadata = map[string]struct {
		Status string `json:"status"`
		Mime   string `json:"m"`
		S      struct {
			U string `json:"u"`
			X int    `json:"x"`
			Y int    `json:"y"`
		} `json:"s"`
	}{
		"img": {Status: "valid", Mime: "image/jpeg", S: struct {
			U string `json:"u"`
			X int    `json:"x"`
			Y int    `json:"y"`
		}{U: "https://i.redd.it/a.jpg", X: 100, Y: 50}},
		"vid": {Status: "valid", Mime: "video/mp4", S: struct {
			U string `json:"u"`
			X int    `json:"x"`
			Y int    `json:"y"`
		}{U: "https://v.redd.it/a.mp4", X: 100, Y: 50}},
	}

	items := listingChildToAlbumMetadata(post, true)
	if len(items) != 1 {
		t.Fatalf("expected 1 image item, got %d", len(items))
	}
}

func listingJSON(after string) string {
	if after == "t3_page2" {
		return `{"data":{"after":"","children":[` + postsJSON(101, 50) + `]}}`
	}
	return `{"data":{"after":"t3_page2","children":[` + postsJSON(1, 100) + `]}}`
}

func postsJSON(start int, count int) string {
	parts := make([]string, 0, count)
	for i := 0; i < count; i++ {
		idx := start + i
		postURL := "https://i.redd.it/image" + strconv.Itoa(idx) + ".jpg"
		postHint := "image"
		gallery := ""
		if idx == 2 || idx == 120 {
			postURL = "https://www.reddit.com/r/wallpaper/comments/example"
			postHint = "link"
		}
		if idx == 3 {
			gallery = `,"is_gallery":true,"gallery_data":{"items":[{"media_id":"m1"},{"media_id":"m2"}]},"media_metadata":{"m1":{"status":"valid","m":"image/jpeg","s":{"u":"https://i.redd.it/gallery1.jpg","x":1200,"y":800}},"m2":{"status":"valid","m":"image/png","s":{"u":"https://i.redd.it/gallery2.png","x":1600,"y":900}}}`
			postURL = "https://www.reddit.com/gallery/example"
			postHint = "link"
		}
		parts = append(parts, `{"kind":"t3","data":{"id":"`+strconv.Itoa(idx)+`","name":"t3_`+strconv.Itoa(idx)+`","author":"user","title":"title","url":"`+postURL+`","thumbnail":"https://i.redd.it/thumb`+strconv.Itoa(idx)+`.jpg","post_hint":"`+postHint+`","over_18":false`+gallery+`,"preview":{"images":[{"source":{"url":"`+postURL+`","width":1920,"height":1080}}]}}}`)
	}
	return strings.Join(parts, ",")
}
