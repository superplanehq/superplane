package mcpserver

import (
	"net/http"
	"testing"
)

func TestPublicOriginUsesLoopbackHost(t *testing.T) {
	req := &http.Request{Host: "localhost:8000"}
	got := PublicOrigin(req, "https://tunnel.example")
	if got != "http://localhost:8000" {
		t.Fatalf("got %q", got)
	}
}

func TestPublicOriginUsesFallbackOffLoopback(t *testing.T) {
	req := &http.Request{Host: "app.example"}
	got := PublicOrigin(req, "https://tunnel.example")
	if got != "https://tunnel.example" {
		t.Fatalf("got %q", got)
	}
}

func TestResourceURL(t *testing.T) {
	if got := ResourceURL("http://localhost:8000/"); got != "http://localhost:8000/mcp" {
		t.Fatalf("got %q", got)
	}
}
