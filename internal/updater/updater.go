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
	releaseURL = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

// Release represents a GitHub release.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
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

// FetchLatestRelease fetches the latest release from GitHub.
func FetchLatestRelease(ctx context.Context) (*Release, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
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

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
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

// AssetNameForPlatform returns the expected asset filename for the current platform.
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
