package mcp

import (
	"net/url"
	"strings"
)

const CodexClientName = "Codex"

// ClientDisplayName returns the label shown for an MCP OAuth client.
// A stored name that is not a Codex metadata URL wins.
// Codex is used when that stored name is empty or is a Codex metadata URL
// and the client id is a Codex metadata URL.
// Otherwise fallback is returned.
func ClientDisplayName(storedName, clientID, fallback string) string {
	stored := strings.TrimSpace(storedName)
	if stored != "" && !isCodexMetadataURL(stored) {
		return stored
	}
	if isCodexMetadataURL(clientID) {
		return CodexClientName
	}
	if stored != "" {
		return stored
	}
	return strings.TrimSpace(fallback)
}

func isCodexMetadataURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") {
		return false
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "chatgpt.com", "www.chatgpt.com":
	default:
		return false
	}
	path := parsed.Path
	return path == "/oauth/codex" || strings.HasPrefix(path, "/oauth/codex/")
}
