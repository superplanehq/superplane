package ws

import "testing"

func TestKindFromTopic(t *testing.T) {
	tests := []struct {
		topic string
		want  string
	}{
		{topic: "agent-session:abc", want: KindAgent},
		{topic: "agent-session:", want: KindAgent},
		{topic: "factory:abc", want: KindFactory},
		{topic: "factory:", want: KindFactory},
		{topic: "11111111-1111-1111-1111-111111111111", want: KindCanvas},
		{topic: "", want: KindCanvas},
		{topic: "agent-sessions:nope", want: KindCanvas},
		{topic: "factories:nope", want: KindCanvas},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			if got := KindFromTopic(tt.topic); got != tt.want {
				t.Fatalf("KindFromTopic(%q) = %q, want %q", tt.topic, got, tt.want)
			}
		})
	}
}
