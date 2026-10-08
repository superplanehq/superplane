package mcp

import "testing"

func TestClientDisplayName(t *testing.T) {
	t.Parallel()

	const codexID = "https://chatgpt.com/oauth/codex/example/client.json"

	tests := []struct {
		name       string
		storedName string
		clientID   string
		fallback   string
		want       string
	}{
		{
			name:     "codex metadata client id",
			clientID: codexID,
			fallback: "MCP client",
			want:     CodexClientName,
		},
		{
			name:       "stored non-url name wins",
			storedName: "Cursor Desktop",
			clientID:   codexID,
			fallback:   codexID,
			want:       "Cursor Desktop",
		},
		{
			name:       "stored codex url does not win",
			storedName: "https://www.ChatGPT.com/oauth/codex",
			clientID:   " HTTPS://chatgpt.com/oauth/codex/example/client.json ",
			fallback:   codexID,
			want:       CodexClientName,
		},
		{
			name:     "exact codex path",
			clientID: "https://chatgpt.com/oauth/codex",
			want:     CodexClientName,
		},
		{
			name:     "codex path prefix",
			clientID: "https://www.chatgpt.com/oauth/codex/client.json",
			want:     CodexClientName,
		},
		{
			name:     "codex lookalike path stays unchanged",
			clientID: "https://chatgpt.com/oauth/codex-evil",
			fallback: "https://chatgpt.com/oauth/codex-evil",
			want:     "https://chatgpt.com/oauth/codex-evil",
		},
		{
			name:     "http codex url stays unchanged",
			clientID: "http://chatgpt.com/oauth/codex/client.json",
			fallback: "http://chatgpt.com/oauth/codex/client.json",
			want:     "http://chatgpt.com/oauth/codex/client.json",
		},
		{
			name:     "other host stays unchanged",
			clientID: "https://chatgpt.com.evil/oauth/codex/client.json",
			fallback: "kept",
			want:     "kept",
		},
		{
			name:     "unrelated https client id stays unchanged",
			clientID: "https://example.com/oauth/client",
			fallback: "https://example.com/oauth/client",
			want:     "https://example.com/oauth/client",
		},
		{
			name:     "empty client uses fallback",
			fallback: "MCP client",
			want:     "MCP client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClientDisplayName(tt.storedName, tt.clientID, tt.fallback)
			if got != tt.want {
				t.Fatalf("ClientDisplayName(%q, %q, %q) = %q, want %q", tt.storedName, tt.clientID, tt.fallback, got, tt.want)
			}
		})
	}
}
