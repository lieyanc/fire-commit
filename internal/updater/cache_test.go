package updater

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNoUpdateInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		channel     string
		consecutive int
		want        time.Duration
	}{
		{name: "latest first", channel: ChannelLatest, consecutive: 1, want: 15 * time.Minute},
		{name: "latest second", channel: ChannelLatest, consecutive: 2, want: 30 * time.Minute},
		{name: "latest third", channel: ChannelLatest, consecutive: 3, want: 60 * time.Minute},
		{name: "latest capped", channel: ChannelLatest, consecutive: 20, want: 12 * time.Hour},
		{name: "stable first", channel: ChannelStable, consecutive: 1, want: 2 * time.Hour},
		{name: "stable second", channel: ChannelStable, consecutive: 2, want: 4 * time.Hour},
		{name: "stable capped", channel: ChannelStable, consecutive: 20, want: 24 * time.Hour},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := noUpdateInterval(tc.channel, tc.consecutive)
			if got != tc.want {
				t.Fatalf("noUpdateInterval(%q, %d) got %v want %v", tc.channel, tc.consecutive, got, tc.want)
			}
		})
	}
}

func TestChannelScheduleTransitions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	state := &channelCheckState{}

	if !shouldCheckChannel(state, now) {
		t.Fatalf("empty state should be checkable")
	}

	recordNoUpdate(state, ChannelLatest, now)
	if state.ConsecutiveNoUpdate != 1 {
		t.Fatalf("consecutive_no_update should be 1, got %d", state.ConsecutiveNoUpdate)
	}
	if state.NextCheckAt.Sub(now) != 15*time.Minute {
		t.Fatalf("next check should be 15m after first no-update")
	}
	if shouldCheckChannel(state, now.Add(10*time.Minute)) {
		t.Fatalf("should not check before next_check_at")
	}

	recordNoUpdate(state, ChannelLatest, now.Add(15*time.Minute))
	if state.ConsecutiveNoUpdate != 2 {
		t.Fatalf("consecutive_no_update should be 2, got %d", state.ConsecutiveNoUpdate)
	}
	if state.NextCheckAt.Sub(now.Add(15*time.Minute)) != 30*time.Minute {
		t.Fatalf("next check should back off to 30m on second no-update")
	}

	recordHasUpdate(state, now.Add(45*time.Minute))
	if state.ConsecutiveNoUpdate != 0 {
		t.Fatalf("consecutive_no_update should reset to 0, got %d", state.ConsecutiveNoUpdate)
	}
	if state.NextCheckAt.Sub(now.Add(45*time.Minute)) != 15*time.Minute {
		t.Fatalf("update-available interval should be 15m")
	}
}

func TestCheckStatePersistence(t *testing.T) {
	t.Parallel()

	p := filepath.Join(t.TempDir(), "state.json")
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	in := &checkState{
		Channels: map[string]*channelCheckState{
			ChannelLatest: {
				ETag:                "etag-1",
				LastSeenVersion:     "dev-10-20260215-abc1234",
				ConsecutiveNoUpdate: 3,
				LastCheckedAt:       now,
				NextCheckAt:         now.Add(time.Hour),
			},
		},
	}
	if err := saveCheckStateToPath(p, in); err != nil {
		t.Fatalf("saveCheckStateToPath() error: %v", err)
	}

	out, err := loadCheckStateFromPath(p)
	if err != nil {
		t.Fatalf("loadCheckStateFromPath() error: %v", err)
	}
	got := out.channel(ChannelLatest)

	if got.ETag != "etag-1" {
		t.Fatalf("etag mismatch: got %q", got.ETag)
	}
	if got.LastSeenVersion != "dev-10-20260215-abc1234" {
		t.Fatalf("last_seen_version mismatch: got %q", got.LastSeenVersion)
	}
	if got.ConsecutiveNoUpdate != 3 {
		t.Fatalf("consecutive_no_update mismatch: got %d", got.ConsecutiveNoUpdate)
	}
	if !got.LastCheckedAt.Equal(now) {
		t.Fatalf("last_checked_at mismatch: got %s want %s", got.LastCheckedAt, now)
	}
	if !got.NextCheckAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("next_check_at mismatch: got %s want %s", got.NextCheckAt, now.Add(time.Hour))
	}
}
