package agent

import "testing"

func TestBrokerStreamURL(t *testing.T) {
	tests := []struct {
		base string
		want string
	}{
		{"http://127.0.0.1:8080", "ws://127.0.0.1:8080/v1/runners/stream"},
		{"http://127.0.0.1:8080/", "ws://127.0.0.1:8080/v1/runners/stream"},
		{"https://broker.example.com", "wss://broker.example.com/v1/runners/stream"},
		{"https://broker.example.com/", "wss://broker.example.com/v1/runners/stream"},
	}
	for _, tt := range tests {
		got, err := brokerStreamURL(tt.base)
		if err != nil {
			t.Fatalf("brokerStreamURL(%q): %v", tt.base, err)
		}
		if got != tt.want {
			t.Errorf("brokerStreamURL(%q) = %q, want %q", tt.base, got, tt.want)
		}
	}
	if _, err := brokerStreamURL(""); err == nil {
		t.Fatal("want error for empty base")
	}
	if _, err := brokerStreamURL("ftp://x"); err == nil {
		t.Fatal("want error for non-http(s) base")
	}
}

func TestTransportWebSocket(t *testing.T) {
	for _, tt := range []struct {
		transport string
		wantWS    bool
	}{
		{"", true},
		{"websocket", true},
		{"WebSocket", true},
		{"HTTP", false},
		{"polling", false},
		{"LEGACY", false},
	} {
		got := transportWebSocket(Config{Transport: tt.transport})
		if got != tt.wantWS {
			t.Errorf("transportWebSocket(%q) = %v, want %v", tt.transport, got, tt.wantWS)
		}
	}
}
