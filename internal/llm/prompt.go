package llm

import (
	"fmt"
	"strings"
)

func buildSystemPrompt(lang string) string {
	langInstruction := "English"
	switch lang {
	case "zh", "zh-CN", "zh-Hans":
		langInstruction = "Simplified Chinese"
	case "zh-TW", "zh-Hant":
		langInstruction = "Traditional Chinese"
	case "ja":
		langInstruction = "Japanese"
	case "ko":
		langInstruction = "Korean"
	case "es":
		langInstruction = "Spanish"
	case "fr":
		langInstruction = "French"
	case "de":
		langInstruction = "German"
	case "ru":
		langInstruction = "Russian"
	default:
		langInstruction = "English"
	}

	return fmt.Sprintf(`You write Git commit messages following Conventional Commits 1.0.0.

Task:
1) Infer the primary intent of the diff.
2) Map it to exactly one commit type.
3) Output one commit header only.

Output format (exactly one line):
<type>(<scope>)!: <description>
or
<type>: <description>

Allowed types:
feat fix docs style refactor perf test build ci chore revert

Type selection rubric (pick the first rule that matches the primary intent):
- revert: explicitly undoes a previous commit/change
- feat: adds a new user-visible capability, API, CLI option, or workflow
- fix: corrects incorrect behavior, bug, regression, crash, or security issue
- perf: improves performance characteristics without changing intended behavior
- refactor: restructures existing code without changing externally observable behavior
- docs: documentation-only changes
- test: test-only changes
- build: build system, dependency, packaging, or toolchain changes
- ci: CI/CD pipeline, workflow, or automation config changes
- style: formatting/lint/whitespace-only changes
- chore: repository maintenance not covered above and not user-visible

Conflict resolution:
- Mixed changes: choose the highest-impact primary intent, not the noisiest file count
- If behavior is restored/corrected, prefer fix over refactor
- If new capability is introduced, prefer feat even if refactor/tests/docs are included
- Do not use docs/test/style when production code behavior also changes
- Use "!" only for breaking changes to public behavior/contracts

Writing rules:
- Scope is optional; include it only when a clear module/component exists
- Description must be imperative and concise, with no trailing period
- For Latin-script languages, start lowercase except proper nouns/acronyms
- Target <= 50 characters; hard limit <= 96 characters
- Focus on intent/outcome, not file-by-file listing
- Output raw message only: no quotes, no markdown, no extra lines
- Write in %s

Good examples:
- feat(auth): add OAuth2 login flow
- fix(config): handle empty env var fallback
- perf(cache): reduce allocations in key lookup
- refactor(api): split handler into service layer
- build(deps): bump go-openai to v1.42.0
- ci(actions): run integration tests on pull request
- feat(api)!: remove legacy v1 endpoints

Bad examples:
- update files
- fix: fix bug
- chore: add new public endpoint
- feat: update README only`, langInstruction)
}

func buildUserPrompt(diff string) string {
	return fmt.Sprintf(`Analyze this git diff and write one Conventional Commit header.

First, silently determine the primary change type using the system rubric.
Then output only one raw commit message line (no quotes, no markdown, no explanation).

Git diff:
%s`, diff)
}

// parseMessage extracts a single commit message from the LLM response.
// It trims whitespace and strips any list prefixes the LLM may have added.
func parseMessage(raw string) string {
	line := strings.TrimSpace(raw)
	if line == "" {
		return ""
	}
	// If multiple lines, take the first non-empty one
	for _, l := range strings.Split(line, "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			return stripPrefix(l)
		}
	}
	return ""
}

func stripPrefix(s string) string {
	// Strip numbered list: "1. ", "2) ", etc.
	for i, c := range s {
		if c >= '0' && c <= '9' {
			continue
		}
		if (c == '.' || c == ')') && i > 0 {
			return strings.TrimSpace(s[i+1:])
		}
		break
	}
	// Strip bullet: "- ", "* "
	if len(s) > 2 && (s[0] == '-' || s[0] == '*') && s[1] == ' ' {
		return strings.TrimSpace(s[2:])
	}
	return s
}
