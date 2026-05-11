package stream

import "testing"

func TestExtractChatID(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/agents/chats/abc-123/stream", "abc-123"},
		{"/agents/chats/abc-123/stream/", "abc-123"},
		{"/agents/chats//stream", ""},
		{"/agents/chats/", ""},
		{"/other/path", ""},
	}

	for _, tt := range tests {
		got := extractChatID(tt.path)
		if got != tt.expected {
			t.Errorf("extractChatID(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}
