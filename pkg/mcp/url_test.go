package mcp

import "testing"

func TestValidatePublicHTTPSURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "public https", raw: "https://api.mobbin.com/mcp"},
		{name: "empty", raw: "", wantErr: true},
		{name: "http", raw: "http://api.mobbin.com/mcp", wantErr: true},
		{name: "loopback", raw: "https://127.0.0.1/mcp", wantErr: true},
		{name: "localhost", raw: "https://localhost/mcp", wantErr: true},
		{name: "private ipv4", raw: "https://10.0.0.8/mcp", wantErr: true},
		{name: "link local", raw: "https://169.254.1.1/mcp", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePublicHTTPSURL(tt.raw)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for %q", tt.raw)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.raw, err)
			}
		})
	}
}
