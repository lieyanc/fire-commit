package main

import (
	"testing"

	"github.com/lieyanc/fire-commit/internal/config"
)

func TestAutoUpdateMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
		cfg     *config.Config
		cfgErr  bool
		want    string
	}{
		{
			name:    "dev build always auto updates",
			version: "dev-1-20260215-abc1234",
			cfg:     &config.Config{AutoUpdate: "n"},
			want:    "a",
		},
		{
			name:    "non dev default notify when config missing",
			version: "v1.2.3",
			cfgErr:  true,
			want:    "y",
		},
		{
			name:    "non dev respects config always",
			version: "v1.2.3",
			cfg:     &config.Config{AutoUpdate: "a"},
			want:    "a",
		},
		{
			name:    "non dev respects config no",
			version: "v1.2.3",
			cfg:     &config.Config{AutoUpdate: "n"},
			want:    "n",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var err error
			if tc.cfgErr {
				err = assertErr{}
			}

			got := autoUpdateMode(tc.version, tc.cfg, err)
			if got != tc.want {
				t.Fatalf("autoUpdateMode(%q) got %q want %q", tc.version, got, tc.want)
			}
		})
	}
}

func TestShouldSkipAutoCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "update command", args: []string{"update"}, want: true},
		{name: "rollback command", args: []string{"rollback"}, want: true},
		{name: "flag then update", args: []string{"--verbose", "update"}, want: true},
		{name: "config command", args: []string{"config"}, want: false},
		{name: "only help flag", args: []string{"--help"}, want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldSkipAutoCheck(tc.args)
			if got != tc.want {
				t.Fatalf("shouldSkipAutoCheck(%v) got %v want %v", tc.args, got, tc.want)
			}
		})
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "test err" }
