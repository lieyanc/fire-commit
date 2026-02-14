package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	repoOwner = "lieyanc"
	repoName  = "fire-commit"
	apiBase   = "https://api.github.com/repos/" + repoOwner + "/" + repoName

	// ChannelLatest includes dev pre-releases and stable releases.
	ChannelLatest = "latest"
	// ChannelStable only includes stable (non-pre-release) releases.
	ChannelStable = "stable"
)

// Release represents a GitHub release.
type Release struct {
	TagName    string  `json:"tag_name"`
	Name       string  `json:"name"`
	Prerelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
}

// Version returns the version string for this release.
// For dev pre-releases (tag "dev"), returns the release Name (e.g. "dev-20260214-abc1234").
// For stable releases, returns the TagName.
func (r *Release) Version() string {
	if r.Prerelease && r.TagName == "dev" {
		return r.Name
	}
	return r.TagName
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckResult holds the result of an update check.
type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	HasUpdate      bool
	Err            error
}

// FetchLatestRelease fetches the latest release from GitHub based on the channel.
// For "stable", it fetches /releases/latest (GitHub excludes pre-releases).
// For "latest", it fetches /releases?per_page=10 and returns the first element.
func FetchLatestRelease(ctx context.Context, channel string) (*Release, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var url string
	switch channel {
	case ChannelStable:
		url = apiBase + "/releases/latest"
	default:
		url = apiBase + "/releases?per_page=10"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	if channel == ChannelStable {
		var release Release
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return nil, err
		}
		return &release, nil
	}

	// Latest channel: decode as array, return first
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}
	return &releases[0], nil
}

// HasNewerVersion checks if the latest version is newer than the current version.
// For stable channel, uses semver comparison.
// For latest channel, uses string inequality (any difference means update available).
// Returns false if current is "dev" (local development build).
func HasNewerVersion(current, latest, channel string) bool {
	if current == "dev" || latest == "" {
		return false
	}
	if channel == ChannelStable {
		return CompareVersions(current, latest)
	}
	// Latest channel: any difference means an update is available
	return current != latest
}

// CompareVersions compares two semver version strings.
// Returns true if latest is newer than current.
// Non-semver current versions (e.g. "dev") cause it to return false.
func CompareVersions(current, latest string) bool {
	cur := parseVersion(current)
	lat := parseVersion(latest)
	if cur == nil || lat == nil {
		return false
	}

	if lat[0] != cur[0] {
		return lat[0] > cur[0]
	}
	if lat[1] != cur[1] {
		return lat[1] > cur[1]
	}
	return lat[2] > cur[2]
}

// parseVersion strips a "v" prefix and parses "major.minor.patch".
// Returns nil if the string is not a valid semver.
func parseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	nums := make([]int, 3)
	for i, p := range parts {
		// Strip any pre-release suffix (e.g. "1-rc1")
		if idx := strings.IndexByte(p, '-'); idx >= 0 {
			p = p[:idx]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	return nums
}

// FindAssetForPlatform finds a release asset matching the current platform by suffix.
// It looks for assets ending in _{os}_{arch}.tar.gz or _{os}_{arch}.zip.
func FindAssetForPlatform(assets []Asset) *Asset {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	suffix := fmt.Sprintf("_%s_%s.%s", goos, goarch, ext)
	for i := range assets {
		if strings.HasSuffix(assets[i].Name, suffix) {
			return &assets[i]
		}
	}
	return nil
}

// AssetNameForPlatform returns the expected asset filename for the current platform.
// Deprecated: use FindAssetForPlatform instead.
func AssetNameForPlatform(version string) string {
	version = strings.TrimPrefix(version, "v")
	os := runtime.GOOS
	arch := runtime.GOARCH

	ext := "tar.gz"
	if os == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("fire-commit_%s_%s_%s.%s", version, os, arch, ext)
}
