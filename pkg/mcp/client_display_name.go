package mcp

import (
	"net/url"
	"strings"
)

const (
	CodexClientName    = "Codex"
	OpenCodeClientName = "OpenCode"
)

// ClientDisplayName returns the label shown for an MCP OAuth client.
// A stored name that is not a known metadata URL wins.
// A known client label is used when that stored name is empty or is a
// metadata URL and the client id is a metadata URL for that client.
// Otherwise fallback is returned.
func ClientDisplayName(storedName, clientID, fallback string) string {
	stored := strings.TrimSpace(storedName)
	if stored != "" && metadataClientName(stored) == "" {
		return stored
	}
	if name := metadataClientName(clientID); name != "" {
		return name
	}
	if stored != "" {
		return stored
	}
	return strings.TrimSpace(fallback)
}

func metadataClientName(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") {
		return ""
	}
	path := parsed.Path
	switch strings.ToLower(parsed.Hostname()) {
	case "chatgpt.com", "www.chatgpt.com":
		if path == "/oauth/codex" || strings.HasPrefix(path, "/oauth/codex/") {
			return CodexClientName
		}
	case "opencode.ai", "www.opencode.ai":
		if path == "/oauth/opencode" || strings.HasPrefix(path, "/oauth/opencode/") {
			return OpenCodeClientName
		}
	}
	return ""
}
