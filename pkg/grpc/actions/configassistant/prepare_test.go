package configassistant

import "testing"

func TestBuildConfigAssistantSuggestURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "spaces only", in: "   ", want: ""},
		{name: "no trailing slash", in: "http://localhost:8090", want: "http://localhost:8090/config-assistant/suggest"},
		{name: "trailing slash", in: "http://localhost:8090/", want: "http://localhost:8090/config-assistant/suggest"},
		{name: "trim spaces", in: "  http://agent:8090/  ", want: "http://agent:8090/config-assistant/suggest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := BuildConfigAssistantSuggestURL(tt.in); got != tt.want {
				t.Fatalf("BuildConfigAssistantSuggestURL(%q) = %q; want %q", tt.in, got, tt.want)
			}
		})
	}
}
