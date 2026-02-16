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
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Version returns the version string for this release.
// For dev pre-releases (tag "dev"), returns the release Name
// (e.g. "dev-1234-20260214-abc1234").
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
// For "latest", it fetches both stable and dev endpoints and selects the
// newest published release regardless of stable/pre-release.
func FetchLatestRelease(ctx context.Context, channel string) (*Release, error) {
	release, _, notModified, err := FetchLatestReleaseConditional(ctx, channel, "")
	if err != nil {
		return nil, err
	}
	if notModified || release == nil {
		return nil, fmt.Errorf("no release payload available")
	}
	return release, nil
}

// FetchLatestReleaseConditional fetches the latest release using optional ETag
// conditional requests. If etag is non-empty, it is sent via If-None-Match.
// It returns notModified=true when GitHub responds with HTTP 304.
func FetchLatestReleaseConditional(ctx context.Context, channel, etag string) (*Release, string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	if channel == ChannelStable {
		release, responseETag, _, status, err := fetchReleaseByURL(ctx, apiBase+"/releases/latest", etag)
		if err != nil {
			return nil, "", false, err
		}
		if status == http.StatusNotModified {
			return nil, responseETag, true, nil
		}
		if status != http.StatusOK {
			return nil, responseETag, false, fmt.Errorf("GitHub API returned status %d", status)
		}
		if release == nil {
			return nil, responseETag, false, fmt.Errorf("no release payload available")
		}
		return release, responseETag, false, nil
	}

	return fetchLatestChannelRelease(ctx, etag)
}

type latestChannelETags struct {
	Stable string `json:"stable,omitempty"`
	Dev    string `json:"dev,omitempty"`
}

func fetchLatestChannelRelease(ctx context.Context, etag string) (*Release, string, bool, error) {
	cached := parseLatestChannelETags(etag)

	stableURL := apiBase + "/releases/latest"
	devURL := apiBase + "/releases/tags/dev"

	stableRelease, stableETag, stableNotModified, stableStatus, err := fetchReleaseByURL(ctx, stableURL, cached.Stable)
	if err != nil {
		return nil, "", false, err
	}
	if stableStatus != http.StatusOK && stableStatus != http.StatusNotModified {
		return nil, stableETag, false, fmt.Errorf("GitHub API returned status %d for stable endpoint", stableStatus)
	}

	devRelease, devETag, devNotModified, devStatus, err := fetchReleaseByURL(ctx, devURL, cached.Dev)
	if err != nil {
		return nil, "", false, err
	}
	switch devStatus {
	case http.StatusOK, http.StatusNotModified, http.StatusNotFound:
	default:
		return nil, devETag, false, fmt.Errorf("GitHub API returned status %d for dev endpoint", devStatus)
	}

	if stableETag == "" {
		stableETag = cached.Stable
	}
	if devETag == "" {
		devETag = cached.Dev
	}
	responseETag := formatLatestChannelETags(stableETag, devETag)

	// Both endpoints unchanged (or dev endpoint absent before and now).
	if stableNotModified && (devNotModified || (devStatus == http.StatusNotFound && cached.Dev == "")) {
		return nil, responseETag, true, nil
	}

	// If only one endpoint changed, the 304 side has no payload. Re-fetch that side
	// without If-None-Match so we can compare published_at accurately.
	if stableStatus == http.StatusNotModified {
		stableRelease, stableETag, _, stableStatus, err = fetchReleaseByURL(ctx, stableURL, "")
		if err != nil {
			return nil, "", false, err
		}
		if stableStatus != http.StatusOK {
			return nil, stableETag, false, fmt.Errorf("GitHub API returned status %d for stable endpoint", stableStatus)
		}
	}
	if devStatus == http.StatusNotModified {
		devRelease, devETag, _, devStatus, err = fetchReleaseByURL(ctx, devURL, "")
		if err != nil {
			return nil, "", false, err
		}
		switch devStatus {
		case http.StatusOK, http.StatusNotFound:
		default:
			return nil, devETag, false, fmt.Errorf("GitHub API returned status %d for dev endpoint", devStatus)
		}
	}

	if stableETag == "" {
		stableETag = cached.Stable
	}
	if devETag == "" {
		devETag = cached.Dev
	}
	responseETag = formatLatestChannelETags(stableETag, devETag)

	latest := newerPublishedRelease(stableRelease, devRelease)
	if latest == nil {
		return nil, responseETag, false, fmt.Errorf("no releases found")
	}
	return latest, responseETag, false, nil
}

func fetchReleaseByURL(ctx context.Context, url, etag string) (*Release, string, bool, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", false, 0, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", false, 0, err
	}
	defer resp.Body.Close()

	responseETag := resp.Header.Get("ETag")
	if resp.StatusCode == http.StatusNotModified {
		return nil, responseETag, true, http.StatusNotModified, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, responseETag, false, resp.StatusCode, nil
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, responseETag, false, http.StatusOK, err
	}
	if release.Draft {
		return nil, responseETag, false, http.StatusNotFound, nil
	}
	return &release, responseETag, false, http.StatusOK, nil
}

func parseLatestChannelETags(raw string) latestChannelETags {
	var out latestChannelETags
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func formatLatestChannelETags(stable, dev string) string {
	if stable == "" && dev == "" {
		return ""
	}
	b, err := json.Marshal(latestChannelETags{Stable: stable, Dev: dev})
	if err != nil {
		return ""
	}
	return string(b)
}

func newerPublishedRelease(a, b *Release) *Release {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if a.PublishedAt.IsZero() && !b.PublishedAt.IsZero() {
		return b
	}
	if b.PublishedAt.IsZero() && !a.PublishedAt.IsZero() {
		return a
	}
	if b.PublishedAt.After(a.PublishedAt) {
		return b
	}
	return a
}

// IsDevVersion returns true for local dev builds ("dev") and CI dev builds
// that use the "dev-*" prefix. Legacy formats are still accepted.
func IsDevVersion(v string) bool {
	return v == "dev" || strings.HasPrefix(v, "dev-")
}

// parseDevVersion parses dev version strings:
//   - Current format:  dev-<build>-<date>-<hash>
//   - Previous format: dev-<date>-<build>-<hash>
//   - Legacy format:   dev-<date>-<hash> (build number treated as 0)
func parseDevVersion(v string) (date string, build int, ok bool) {
	if !IsDevVersion(v) || v == "dev" {
		return "", 0, false
	}

	parts := strings.Split(v, "-")
	if len(parts) < 3 || parts[0] != "dev" {
		return "", 0, false
	}

	// New format: dev-<build>-<date>-<hash>
	if len(parts) >= 4 && isDigits(parts[1]) && isDateToken(parts[2]) {
		n, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, false
		}
		return parts[2], n, true
	}

	// Previous format: dev-<date>-<build>-<hash>
	if len(parts) >= 4 && isDateToken(parts[1]) && isDigits(parts[2]) {
		n, err := strconv.Atoi(parts[2])
		if err != nil {
			return "", 0, false
		}
		return parts[1], n, true
	}

	// Legacy format: dev-<date>-<hash>
	if isDateToken(parts[1]) {
		return parts[1], 0, true
	}

	return "", 0, false
}

func isDateToken(s string) bool {
	return len(s) == 8 && isDigits(s)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// HasNewerVersion checks if the latest version is newer than the current version.
//   - Same version string: no update.
//   - Local "dev" build: any published release is newer.
//   - Stable channel: semver comparison, with dev->stable migration.
//   - Latest channel with two dev versions: build-number + date comparison.
//   - Latest channel with mixed dev/stable: always update to follow newest release stream.
func HasNewerVersion(current, latest, channel string) bool {
	if latest == "" || current == latest {
		return false
	}
	if current == "dev" {
		return true
	}
	currentIsDev := IsDevVersion(current)
	latestIsDev := IsDevVersion(latest)

	if channel == ChannelStable {
		if currentIsDev && !latestIsDev {
			return true
		}
		return compareSemverWithRecovery(current, latest)
	}

	// Latest channel: dev versions use build number + date comparison.
	if currentIsDev && latestIsDev {
		curDate, curBuild, curOK := parseDevVersion(current)
		latDate, latBuild, latOK := parseDevVersion(latest)
		if curOK && latOK {
			// Build number is the primary ordering key.
			if latBuild != curBuild {
				return latBuild > curBuild
			}
			// Date is a secondary tie-breaker.
			return latDate > curDate
		}
		// If the current version is unparsable but latest is valid dev format,
		// prefer upgrading to recover to a known format.
		if latOK && !curOK {
			return true
		}
		if curDate != "" && latDate != "" {
			return latDate > curDate
		}
		// Two unparsable dev strings: prefer updating to recover to published head.
		return true
	}

	// Latest channel includes both stable and dev releases. If release type changed,
	// follow the newest published stream.
	if currentIsDev != latestIsDev {
		return true
	}

	return compareSemverWithRecovery(current, latest)
}

func compareSemverWithRecovery(current, latest string) bool {
	if CompareVersions(current, latest) {
		return true
	}
	cur := parseVersion(current)
	lat := parseVersion(latest)
	// If current is unparsable but latest is valid semver, recover by upgrading.
	return cur == nil && lat != nil
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
