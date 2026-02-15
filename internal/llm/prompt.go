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

Output exactly one line:
<type>(<scope>)!: <description>
or
<type>: <description>

Allowed types:
feat fix docs style refactor perf test build ci chore revert

Rules:
- Scope is optional; include it only when a clear module/component exists
- Use "!" only for breaking changes
- Description must be imperative and concise, with no trailing period
- For Latin-script languages, start lowercase except proper nouns/acronyms
- Target <= 50 characters; hard limit <= 96 characters
- Focus on intent or outcome, not a file-by-file listing
- Output raw message only: no quotes, no list markers, no extra lines
- Write in %s

Good examples:
- feat(auth): add OAuth2 login flow
- fix: prevent nil pointer on empty config
- refactor(api): split handler into service layer
- feat(api)!: remove legacy v1 endpoints

Bad examples:
- update files (too vague)
- fix: fix the bug (redundant)
- feat: add function handleAuth and modify config.go and update tests (file listing)`, langInstruction)
}

func buildUserPrompt(diff string) string {
	return fmt.Sprintf(`Write one commit message for this diff. Output the raw message only — no quotes, no markup, no explanation.

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
