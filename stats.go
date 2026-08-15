package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var immichUserAgent = "trmnl-immich-stats/" + Version

type immichClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func newImmichClient(baseURL, apiKey string) *immichClient {
	return &immichClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *immichClient) do(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, err
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", immichUserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.http.Do(req)
}

func (c *immichClient) getJSON(ctx context.Context, path string, dst any) error {
	resp, err := c.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

// Upstream DTOs (subsets of the Immich OpenAPI spec).

type serverStatistics struct {
	Photos      int64         `json:"photos"`
	Videos      int64         `json:"videos"`
	Usage       int64         `json:"usage"`
	UsageByUser []usageByUser `json:"usageByUser"`
	UsagePhotos int64         `json:"usagePhotos"`
	UsageVideos int64         `json:"usageVideos"`
}

type usageByUser struct {
	Photos           int64  `json:"photos"`
	Videos           int64  `json:"videos"`
	Usage            int64  `json:"usage"`
	UsagePhotos      int64  `json:"usagePhotos"`
	UsageVideos      int64  `json:"usageVideos"`
	QuotaSizeInBytes int64  `json:"quotaSizeInBytes"`
	UserID           string `json:"userId"`
	UserName         string `json:"userName"`
}

type serverVersion struct {
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	Prerelease string `json:"prerelease"`
}

type serverAbout struct {
	Version       string `json:"version"`
	Build         string `json:"build"`
	BuildImage    string `json:"buildImage"`
	BuildImageURL string `json:"buildImageUrl"`
	BuildURL      string `json:"buildUrl"`
	ExifTool      string `json:"exiftool"`
	FFmpeg        string `json:"ffmpeg"`
	ImageMagick   string `json:"imagemagick"`
	Libvips       string `json:"libvips"`
	Licensed      bool   `json:"licensed"`
	NodeJS        string `json:"nodejs"`
	Repository    string `json:"repository"`
	RepositoryURL string `json:"repositoryUrl"`
	SourceCommit  string `json:"sourceCommit"`
	SourceRef     string `json:"sourceRef"`
	SourceURL     string `json:"sourceUrl"`
}

// TRMNL-pollable payload.

type userStats struct {
	UserName   string `json:"user_name"`
	Photos     int64  `json:"photos"`
	Videos     int64  `json:"videos"`
	Usage      int64  `json:"usage"`
	UsageHuman string `json:"usage_human"`
	Quota      int64  `json:"quota"`
	QuotaHuman string `json:"quota_human"`
}

type serverInfo struct {
	Version       string `json:"version"`
	Build         string `json:"build"`
	Licensed      bool   `json:"licensed"`
	NodeJS        string `json:"nodejs"`
	FFmpeg        string `json:"ffmpeg"`
	ExifTool      string `json:"exiftool"`
	ImageMagick   string `json:"imagemagick"`
	Libvips       string `json:"libvips"`
	SourceCommit  string `json:"source_commit"`
	SourceURL     string `json:"source_url"`
	RepositoryURL string `json:"repository_url"`
}

type stats struct {
	Photos      int64         `json:"photos"`
	Videos      int64         `json:"videos"`
	TotalAssets int64         `json:"total_assets"`
	Usage       int64         `json:"usage"`
	UsageHuman  string        `json:"usage_human"`
	UsageByUser []userStats   `json:"usage_by_user"`
	VersionFull string        `json:"version_full"`
	Version     serverVersion `json:"version"`
	Server      serverInfo    `json:"server"`
	License     any           `json:"license"`
}

// formatBytes renders a byte count with two significant figures.
func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func (c *immichClient) collectStats(ctx context.Context) (*stats, error) {
	var stat serverStatistics
	if err := c.getJSON(ctx, "/api/server/statistics", &stat); err != nil {
		return nil, fmt.Errorf("statistics: %w", err)
	}
	var ver serverVersion
	if err := c.getJSON(ctx, "/api/server/version", &ver); err != nil {
		return nil, fmt.Errorf("version: %w", err)
	}
	var about serverAbout
	if err := c.getJSON(ctx, "/api/server/about", &about); err != nil {
		return nil, fmt.Errorf("about: %w", err)
	}
	var license any
	if err := c.getJSON(ctx, "/api/server/license", &license); err != nil {
		return nil, fmt.Errorf("license: %w", err)
	}

	users := make([]userStats, 0, len(stat.UsageByUser))
	for _, u := range stat.UsageByUser {
		users = append(users, userStats{
			UserName:   u.UserName,
			Photos:     u.Photos,
			Videos:     u.Videos,
			Usage:      u.Usage,
			UsageHuman: formatBytes(u.Usage),
			Quota:      u.QuotaSizeInBytes,
			QuotaHuman: formatBytes(u.QuotaSizeInBytes),
		})
	}

	versionFull := fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Patch)
	if ver.Prerelease != "" {
		versionFull += "-" + ver.Prerelease
	}

	return &stats{
		Photos:      stat.Photos,
		Videos:      stat.Videos,
		TotalAssets: stat.Photos + stat.Videos,
		Usage:       stat.Usage,
		UsageHuman:  formatBytes(stat.Usage),
		UsageByUser: users,
		VersionFull: versionFull,
		Version:     ver,
		Server: serverInfo{
			Version:       about.Version,
			Build:         about.Build,
			Licensed:      about.Licensed,
			NodeJS:        about.NodeJS,
			FFmpeg:        about.FFmpeg,
			ExifTool:      about.ExifTool,
			ImageMagick:   about.ImageMagick,
			Libvips:       about.Libvips,
			SourceCommit:  about.SourceCommit,
			SourceURL:     about.SourceURL,
			RepositoryURL: about.RepositoryURL,
		},
		License: license,
	}, nil
}

func (c *immichClient) handleStats(w http.ResponseWriter, r *http.Request) {
	s, err := c.collectStats(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}
