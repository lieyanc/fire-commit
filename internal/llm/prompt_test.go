package llm

import (
	"strings"
	"testing"
)

func TestBuildSystemPromptIncludesTypeSelectionRubric(t *testing.T) {
	got := buildSystemPrompt("en")
	required := []string{
		"Type selection rubric",
		"Conflict resolution",
		"- feat: adds a new user-visible capability",
		"- fix: corrects incorrect behavior",
		"- refactor: restructures existing code",
		"- build: build system, dependency",
		"- ci: CI/CD pipeline, workflow, or automation config changes",
	}
	for _, r := range required {
		if !strings.Contains(got, r) {
			t.Fatalf("system prompt missing %q", r)
		}
	}
}

func TestBuildSystemPromptLanguageMapping(t *testing.T) {
	got := buildSystemPrompt("zh")
	if !strings.Contains(got, "Write in Simplified Chinese") {
		t.Fatalf("expected simplified Chinese instruction, got: %q", got)
	}
}

func TestBuildUserPromptRequestsSilentClassification(t *testing.T) {
	diff := "diff --git a/main.go b/main.go\n+func main() {}"
	got := buildUserPrompt(diff)

	if !strings.Contains(got, "silently determine the primary change type") {
		t.Fatalf("user prompt missing classification instruction")
	}
	if !strings.Contains(got, "Then output only one raw commit message line") {
		t.Fatalf("user prompt missing output constraint")
	}
	if !strings.Contains(got, diff) {
		t.Fatalf("user prompt missing injected diff")
	}
}

func TestParseMessage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "fix: prevent nil pointer", want: "fix: prevent nil pointer"},
		{name: "numbered", in: "1. feat: add login page", want: "feat: add login page"},
		{name: "bullet", in: "- refactor: split service layer", want: "refactor: split service layer"},
		{name: "first non-empty line", in: "\n\nchore: bump version\nextra", want: "chore: bump version"},
		{name: "empty", in: " \n\t", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseMessage(tc.in)
			if got != tc.want {
				t.Fatalf("parseMessage() = %q, want %q", got, tc.want)
			}
		})
	}
}

