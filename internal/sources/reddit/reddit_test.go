package reddit

import (
	"encoding/json"
	"strings"
	"testing"
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
	}, 25)
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
	}, 10)
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
