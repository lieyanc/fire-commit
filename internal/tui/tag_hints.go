package tui

import (
	"fmt"
	"strconv"
	"strings"
)

type tagHints struct {
	base  string
	minor string
	patch string
}

func buildTagHints(latest string) tagHints {
	const defaultTag = "v1.0.0"

	major, minor, patch, ok := parseTagVersion(latest)
	if !ok {
		major, minor, patch, _ = parseTagVersion(defaultTag)
	}

	return tagHints{
		base:  formatTagVersion(major, minor, patch),
		minor: formatTagVersion(major, minor+1, 0),
		patch: formatTagVersion(major, minor, patch+1),
	}
}

func parseTagVersion(tag string) (int, int, int, bool) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return 0, 0, 0, false
	}
	if strings.HasPrefix(tag, "v") || strings.HasPrefix(tag, "V") {
		tag = tag[1:]
	}

	parts := strings.SplitN(tag, ".", 3)
	if len(parts) != 3 {
		return 0, 0, 0, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, false
	}

	patchPart := parts[2]
	if idx := strings.IndexAny(patchPart, "-+"); idx >= 0 {
		patchPart = patchPart[:idx]
	}
	patch, err := strconv.Atoi(patchPart)
	if err != nil {
		return 0, 0, 0, false
	}

	if major < 0 || minor < 0 || patch < 0 {
		return 0, 0, 0, false
	}

	return major, minor, patch, true
}

func formatTagVersion(major, minor, patch int) string {
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}
