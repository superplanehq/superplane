package mcpserver

import (
	"net"
	"net/http"
	"strings"
)

const (
	PathMCP                 = "/mcp"
	PathAuthorize           = "/oauth/authorize"
	PathToken               = "/oauth/token"
	PathRegister            = "/oauth/register"
	PathProtectedResource   = "/.well-known/oauth-protected-resource"
	PathAuthorizationServer = "/.well-known/oauth-authorization-server"

	LocalClientID   = "superplane-local"
	LocalClientName = "Cursor"
	ServerName      = "SuperPlane"
	ServerVersion   = "1.0.0"

	AccessTokenPurpose = "mcp_workspace"
	ConsentPurpose     = "mcp_consent"
)

var CursorRedirectURIs = []string{
	"http://localhost:8787/callback",
	"cursor://anysphere.cursor-mcp/oauth/callback",
}

var GrantedScopes = []string{
	"work_orders:read",
	"work_orders:create",
	"work_orders:update",
}

func PublicOrigin(r *http.Request, fallback string) string {
	host := strings.TrimSpace(r.Host)
	if host == "" && r.URL != nil {
		host = strings.TrimSpace(r.URL.Host)
	}
	if isLoopbackHost(host) {
		return requestScheme(r) + "://" + host
	}
	trimmed := strings.TrimRight(strings.TrimSpace(fallback), "/")
	if trimmed != "" {
		return trimmed
	}
	if host != "" {
		return requestScheme(r) + "://" + host
	}
	return "http://localhost:8000"
}

func ResourceURL(origin string) string {
	return strings.TrimRight(strings.TrimSpace(origin), "/") + PathMCP
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return "https"
	}
	return "http"
}

func isLoopbackHost(hostPort string) bool {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
	}
	lower := strings.ToLower(strings.TrimSpace(host))
	if lower == "" {
		return false
	}
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return true
	}
	ip := net.ParseIP(lower)
	return ip != nil && ip.IsLoopback()
}
