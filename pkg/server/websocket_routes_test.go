package server

import "testing"

func TestShouldRegisterWebSocketRoutes(t *testing.T) {
	tests := []struct {
		name            string
		websocketServer string
		webServer       string
		want            bool
	}{
		{name: "explicit yes", websocketServer: "yes", webServer: "no", want: true},
		{name: "explicit no", websocketServer: "no", webServer: "yes", want: false},
		{name: "unset follows web yes", websocketServer: "", webServer: "yes", want: true},
		{name: "unset follows web no", websocketServer: "", webServer: "no", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("START_WEBSOCKET_SERVER", tt.websocketServer)
			t.Setenv("START_WEB_SERVER", tt.webServer)
			if got := shouldRegisterWebSocketRoutes(); got != tt.want {
				t.Fatalf("shouldRegisterWebSocketRoutes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldRegisterGRPCGateway(t *testing.T) {
	t.Run("unset defaults on", func(t *testing.T) {
		t.Setenv("START_GRPC_GATEWAY", "")
		if !shouldRegisterGRPCGateway() {
			t.Fatal("expected true when unset")
		}
	})
	t.Run("yes", func(t *testing.T) {
		t.Setenv("START_GRPC_GATEWAY", "yes")
		if !shouldRegisterGRPCGateway() {
			t.Fatal("expected true")
		}
	})
	t.Run("no", func(t *testing.T) {
		t.Setenv("START_GRPC_GATEWAY", "no")
		if shouldRegisterGRPCGateway() {
			t.Fatal("expected false")
		}
	})
}
