package tui

import "testing"

func TestBuildTagHintsFromLatest(t *testing.T) {
	t.Parallel()

	hints := buildTagHints("v1.2.3")
	if hints.base != "v1.2.3" {
		t.Fatalf("base got %q want %q", hints.base, "v1.2.3")
	}
	if hints.minor != "v1.3.0" {
		t.Fatalf("minor got %q want %q", hints.minor, "v1.3.0")
	}
	if hints.patch != "v1.2.4" {
		t.Fatalf("patch got %q want %q", hints.patch, "v1.2.4")
	}
}

func TestBuildTagHintsAcceptsVersionWithoutPrefix(t *testing.T) {
	t.Parallel()

	hints := buildTagHints("1.2.3")
	if hints.base != "v1.2.3" {
		t.Fatalf("base got %q want %q", hints.base, "v1.2.3")
	}
	if hints.minor != "v1.3.0" {
		t.Fatalf("minor got %q want %q", hints.minor, "v1.3.0")
	}
	if hints.patch != "v1.2.4" {
		t.Fatalf("patch got %q want %q", hints.patch, "v1.2.4")
	}
}

func TestBuildTagHintsFallbackDefault(t *testing.T) {
	t.Parallel()

	hints := buildTagHints("dev")
	if hints.base != "v1.0.0" {
		t.Fatalf("base got %q want %q", hints.base, "v1.0.0")
	}
	if hints.minor != "v1.1.0" {
		t.Fatalf("minor got %q want %q", hints.minor, "v1.1.0")
	}
	if hints.patch != "v1.0.1" {
		t.Fatalf("patch got %q want %q", hints.patch, "v1.0.1")
	}
}

func TestResolveTagShortcut(t *testing.T) {
	t.Parallel()

	m := Model{
		tagHintMinor: "v2.0.0",
		tagHintPatch: "v1.9.10",
	}

	if got := m.resolveTagShortcut("+0.1"); got != "v2.0.0" {
		t.Fatalf("+0.1 got %q want %q", got, "v2.0.0")
	}
	if got := m.resolveTagShortcut("+0.01"); got != "v1.9.10" {
		t.Fatalf("+0.01 got %q want %q", got, "v1.9.10")
	}
	if got := m.resolveTagShortcut("v3.0.0"); got != "v3.0.0" {
		t.Fatalf("passthrough got %q want %q", got, "v3.0.0")
	}
}
