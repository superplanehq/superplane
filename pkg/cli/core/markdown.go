package core

import (
	"strings"

	"charm.land/glamour/v2"
)

// RenderMarkdownForTerminal renders markdown as ANSI-formatted terminal text.
// When rendering fails, it falls back to plain markdown text.
func RenderMarkdownForTerminal(markdown string) string {
	markdown = strings.TrimSpace(markdown)
	if markdown == "" {
		return ""
	}

	rendered, err := glamour.RenderWithEnvironmentConfig(markdown)
	if err != nil {
		return markdown
	}

	rendered = strings.TrimSpace(rendered)
	if rendered == "" {
		return markdown
	}

	return rendered
}
