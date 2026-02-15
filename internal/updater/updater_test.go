package updater

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseDevVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    string
		date  string
		build int
		ok    bool
	}{
		{
			name:  "new format",
			in:    "dev-1234-20260215-abc1234",
			date:  "20260215",
			build: 1234,
			ok:    true,
		},
		{
			name:  "previous format",
			in:    "dev-20260215-1234-abc1234",
			date:  "20260215",
			build: 1234,
			ok:    true,
		},
		{
			name:  "legacy format",
			in:    "dev-20260215-abc1234",
			date:  "20260215",
			build: 0,
			ok:    true,
		},
		{
			name: "local dev literal",
			in:   "dev",
			ok:   false,
		},
		{
			name: "invalid build number",
			in:   "dev-notnum-20260215-abc1234",
			ok:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			date, build, ok := parseDevVersion(tc.in)
			if ok != tc.ok {
				t.Fatalf("ok mismatch: got %v want %v", ok, tc.ok)
			}
			if date != tc.date {
				t.Fatalf("date mismatch: got %q want %q", date, tc.date)
			}
			if build != tc.build {
				t.Fatalf("build mismatch: got %d want %d", build, tc.build)
			}
		})
	}
}

func TestHasNewerVersionDev(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{
			name:    "same date higher build",
			current: "dev-10-20260215-aaaaaaa",
			latest:  "dev-11-20260215-bbbbbbb",
			want:    true,
		},
		{
			name:    "same date lower build",
			current: "dev-11-20260215-bbbbbbb",
			latest:  "dev-10-20260215-aaaaaaa",
			want:    false,
		},
		{
			name:    "higher build wins even if date older",
			current: "dev-100-20260216-aaaaaaa",
			latest:  "dev-101-20260215-bbbbbbb",
			want:    true,
		},
		{
			name:    "same build uses date as tie breaker",
			current: "dev-100-20260214-aaaaaaa",
			latest:  "dev-100-20260215-bbbbbbb",
			want:    true,
		},
		{
			name:    "legacy current can upgrade to new format",
			current: "dev-20260215-aaaaaaa",
			latest:  "dev-1-20260215-bbbbbbb",
			want:    true,
		},
		{
			name:    "previous format is still comparable",
			current: "dev-20260215-10-aaaaaaa",
			latest:  "dev-11-20260215-bbbbbbb",
			want:    true,
		},
		{
			name:    "local dev always updates",
			current: "dev",
			latest:  "dev-1-20260215-bbbbbbb",
			want:    true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := HasNewerVersion(tc.current, tc.latest, ChannelLatest)
			if got != tc.want {
				t.Fatalf("HasNewerVersion(%q, %q) got %v want %v", tc.current, tc.latest, got, tc.want)
			}
		})
	}
}

func TestFetchLatestReleaseConditionalUsesETag(t *testing.T) {
	oldClient := http.DefaultClient
	t.Cleanup(func() {
		http.DefaultClient = oldClient
	})

	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got, want := req.Header.Get("If-None-Match"), "\"etag-1\""; got != want {
				t.Fatalf("If-None-Match mismatch: got %q want %q", got, want)
			}
			if req.URL.Path != "/repos/lieyanc/fire-commit/releases" {
				t.Fatalf("unexpected request path: %s", req.URL.Path)
			}
			resp := &http.Response{
				StatusCode: http.StatusNotModified,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    req,
			}
			resp.Header.Set("ETag", "\"etag-2\"")
			return resp, nil
		}),
	}

	release, etag, notModified, err := FetchLatestReleaseConditional(
		context.Background(),
		ChannelLatest,
		"\"etag-1\"",
	)
	if err != nil {
		t.Fatalf("FetchLatestReleaseConditional() error: %v", err)
	}
	if release != nil {
		t.Fatalf("release should be nil on 304")
	}
	if !notModified {
		t.Fatalf("notModified should be true on 304")
	}
	if etag != "\"etag-2\"" {
		t.Fatalf("etag mismatch: got %q want %q", etag, "\"etag-2\"")
	}
}

func TestFetchLatestReleaseConditionalParsesStable(t *testing.T) {
	oldClient := http.DefaultClient
	t.Cleanup(func() {
		http.DefaultClient = oldClient
	})

	http.DefaultClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/repos/lieyanc/fire-commit/releases/latest" {
				t.Fatalf("unexpected request path: %s", req.URL.Path)
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`{"tag_name":"v1.2.3","name":"v1.2.3","prerelease":false,"assets":[]}`,
				)),
				Request: req,
			}
			resp.Header.Set("ETag", "\"etag-stable\"")
			return resp, nil
		}),
	}

	release, etag, notModified, err := FetchLatestReleaseConditional(
		context.Background(),
		ChannelStable,
		"",
	)
	if err != nil {
		t.Fatalf("FetchLatestReleaseConditional() error: %v", err)
	}
	if notModified {
		t.Fatalf("notModified should be false on 200")
	}
	if etag != "\"etag-stable\"" {
		t.Fatalf("etag mismatch: got %q want %q", etag, "\"etag-stable\"")
	}
	if release == nil || release.TagName != "v1.2.3" {
		t.Fatalf("unexpected release: %#v", release)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
