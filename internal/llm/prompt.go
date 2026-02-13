package llm

import "fmt"

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

func buildUserPrompt(diff string, n int) string {
	return fmt.Sprintf(`Based on the following git diff, generate exactly %d different commit message suggestions. Each message should capture the essence of the changes from a different angle or level of detail.

Output ONLY the commit messages, one per line, with no numbering, bullets, or extra text.

Git diff:
%s`, n, diff)
}
