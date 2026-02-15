package updater

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
)

const (
	checkIntervalUpdateAvailable = 15 * time.Minute

	checkIntervalLatestBase = 15 * time.Minute
	checkIntervalLatestMax  = 12 * time.Hour

	checkIntervalStableBase = 2 * time.Hour
	checkIntervalStableMax  = 24 * time.Hour

	checkRetryLatest = 15 * time.Minute
	checkRetryStable = 60 * time.Minute
)

// checkState is persisted to disk to support adaptive scheduling and
// conditional requests with ETag.
type checkState struct {
	Channels map[string]*channelCheckState `json:"channels"`
}

type channelCheckState struct {
	ETag                string    `json:"etag,omitempty"`
	LastSeenVersion     string    `json:"last_seen_version,omitempty"`
	ConsecutiveNoUpdate int       `json:"consecutive_no_update,omitempty"`
	NextCheckAt         time.Time `json:"next_check_at,omitempty"`
	LastCheckedAt       time.Time `json:"last_checked_at,omitempty"`
}

func updateStatePath() string {
	return filepath.Join(xdg.CacheHome, "firecommit", "update-check-state.json")
}

func loadCheckState() (*checkState, error) {
	return loadCheckStateFromPath(updateStatePath())
}

func loadCheckStateFromPath(path string) (*checkState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &checkState{}, nil
		}
		return nil, err
	}

	var state checkState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state.Channels == nil {
		state.Channels = make(map[string]*channelCheckState)
	}
	return &state, nil
}

func saveCheckState(state *checkState) error {
	return saveCheckStateToPath(updateStatePath(), state)
}

func saveCheckStateToPath(path string, state *checkState) error {
	p := path
	_ = os.MkdirAll(filepath.Dir(p), 0o755)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func (s *checkState) channel(channel string) *channelCheckState {
	if s.Channels == nil {
		s.Channels = make(map[string]*channelCheckState)
	}
	cs, ok := s.Channels[channel]
	if !ok || cs == nil {
		cs = &channelCheckState{}
		s.Channels[channel] = cs
	}
	return cs
}

func shouldCheckChannel(state *channelCheckState, now time.Time) bool {
	if state == nil || state.NextCheckAt.IsZero() {
		return true
	}
	return !now.Before(state.NextCheckAt)
}

func recordNoUpdate(state *channelCheckState, channel string, now time.Time) {
	if state == nil {
		return
	}
	state.LastCheckedAt = now
	state.ConsecutiveNoUpdate++
	state.NextCheckAt = now.Add(noUpdateInterval(channel, state.ConsecutiveNoUpdate))
}

func recordHasUpdate(state *channelCheckState, now time.Time) {
	if state == nil {
		return
	}
	state.LastCheckedAt = now
	state.ConsecutiveNoUpdate = 0
	state.NextCheckAt = now.Add(checkIntervalUpdateAvailable)
}

func recordFetchError(state *channelCheckState, channel string, now time.Time) {
	if state == nil {
		return
	}
	state.LastCheckedAt = now
	state.NextCheckAt = now.Add(errorRetryInterval(channel))
}

func noUpdateInterval(channel string, consecutiveNoUpdate int) time.Duration {
	base, max := intervalBoundsForChannel(channel)
	if consecutiveNoUpdate <= 0 {
		consecutiveNoUpdate = 1
	}

	exp := consecutiveNoUpdate - 1
	if exp > 16 {
		exp = 16
	}

	interval := base * time.Duration(1<<exp)
	if interval > max {
		return max
	}
	return interval
}

func errorRetryInterval(channel string) time.Duration {
	if channel == ChannelStable {
		return checkRetryStable
	}
	return checkRetryLatest
}

func intervalBoundsForChannel(channel string) (base, max time.Duration) {
	if channel == ChannelStable {
		return checkIntervalStableBase, checkIntervalStableMax
	}
	return checkIntervalLatestBase, checkIntervalLatestMax
}
