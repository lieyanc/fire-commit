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

	return fmt.Sprintf(`You are an expert at writing concise, meaningful git commit messages following the Conventional Commits specification.

Rules:
1. Use the format: type(scope): description
2. Types: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
3. Scope is optional but encouraged when clear
4. Description should be lowercase, imperative mood, no period at end
5. Keep the first line under 72 characters
6. Write the commit message in %s

Analyze the provided git diff and generate commit messages that accurately describe the changes.`, langInstruction)
}

func buildUserPrompt(diff string) string {
	return fmt.Sprintf(`Based on the following git diff, generate exactly one commit message that accurately describes the changes.

Output ONLY the commit message on a single line, with no numbering, bullets, or extra text.

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
