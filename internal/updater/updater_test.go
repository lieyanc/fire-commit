package updater

import "testing"

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
			in:   "dev-20260215-notnum-abc1234",
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
			current: "dev-20260215-10-aaaaaaa",
			latest:  "dev-20260215-11-bbbbbbb",
			want:    true,
		},
		{
			name:    "same date lower build",
			current: "dev-20260215-11-bbbbbbb",
			latest:  "dev-20260215-10-aaaaaaa",
			want:    false,
		},
		{
			name:    "newer date",
			current: "dev-20260214-999-aaaaaaa",
			latest:  "dev-20260215-1-bbbbbbb",
			want:    true,
		},
		{
			name:    "legacy current can upgrade to new format",
			current: "dev-20260215-aaaaaaa",
			latest:  "dev-20260215-1-bbbbbbb",
			want:    true,
		},
		{
			name:    "local dev always updates",
			current: "dev",
			latest:  "dev-20260215-1-bbbbbbb",
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
