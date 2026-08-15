package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testKey = "test-api-key"

func fakeImmich(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != testKey {
			t.Errorf("expected x-api-key %q, got %q", testKey, r.Header.Get("x-api-key"))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/server/statistics":
			_ = json.NewEncoder(w).Encode(serverStatistics{
				Photos: 1200,
				Videos: 300,
				Usage:  1536000000,
				UsageByUser: []usageByUser{
					{UserName: "alice", Photos: 1200, Videos: 300, Usage: 1536000000, QuotaSizeInBytes: 0},
				},
				UsagePhotos: 1024000000,
				UsageVideos: 512000000,
			})
		case "/api/server/version":
			_ = json.NewEncoder(w).Encode(serverVersion{Major: 1, Minor: 132, Patch: 0, Prerelease: ""})
		case "/api/server/about":
			_ = json.NewEncoder(w).Encode(serverAbout{
				Version:       "v1.132.0",
				Build:         "2026-08-01",
				Licensed:      true,
				NodeJS:        "22.14.0",
				FFmpeg:        "7.1.1",
				ExifTool:      "13.20",
				ImageMagick:   "7.1.1-43",
				Libvips:       "8.16.1",
				SourceCommit:  "abc123",
				SourceURL:     "https://github.com/immich-app/immich/commit/abc123",
				RepositoryURL: "https://github.com/immich-app/immich",
			})
		case "/api/server/license":
			_ = json.NewEncoder(w).Encode(map[string]any{"licenseKey": "lkey-123", "activationKey": "akey-456", "activatedAt": "2026-01-01T00:00:00Z"})
		default:
			http.NotFound(w, r)
		}
	}))
}

func newTestClient(t *testing.T, server *httptest.Server) *immichClient {
	t.Helper()
	return newImmichClient(server.URL, testKey)
}

func TestCollectStats(t *testing.T) {
	server := fakeImmich(t)
	defer server.Close()
	client := newTestClient(t, server)

	s, err := client.collectStats(t.Context())
	if err != nil {
		t.Fatalf("collectStats: %v", err)
	}
	if s.Photos != 1200 || s.Videos != 300 {
		t.Errorf("expected 1200 photos / 300 videos, got %d / %d", s.Photos, s.Videos)
	}
	if s.TotalAssets != 1500 {
		t.Errorf("expected 1500 total assets, got %d", s.TotalAssets)
	}
	if s.UsageHuman != "1.4 GB" {
		t.Errorf("expected 1.4 GB, got %q", s.UsageHuman)
	}
	if s.VersionFull != "1.132.0" {
		t.Errorf("expected 1.132.0, got %q", s.VersionFull)
	}
	if !s.Server.Licensed {
		t.Error("expected licensed true")
	}
	if s.Server.NodeJS != "22.14.0" || s.Server.FFmpeg != "7.1.1" {
		t.Errorf("unexpected server info: %+v", s.Server)
	}
	if len(s.UsageByUser) != 1 || s.UsageByUser[0].UserName != "alice" {
		t.Errorf("unexpected usage by user: %+v", s.UsageByUser)
	}
	if s.UsageByUser[0].UsageHuman != "1.4 GB" {
		t.Errorf("expected user usage 1.4 GB, got %q", s.UsageByUser[0].UsageHuman)
	}
	if s.License == nil {
		t.Error("expected license passthrough")
	}
}

func TestStatsHandler(t *testing.T) {
	server := fakeImmich(t)
	defer server.Close()
	client := newTestClient(t, server)

	req := httptest.NewRequest(http.MethodGet, "/api/trmnl/stats", nil)
	rec := httptest.NewRecorder()
	client.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var s stats
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if s.Photos != 1200 {
		t.Errorf("expected 1200 photos, got %d", s.Photos)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1536000000, "1.4 GB"},
		{1099511627776, "1.0 TB"},
	}
	for _, tc := range cases {
		if got := formatBytes(tc.in); got != tc.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
