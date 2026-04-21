package public

import (
	"net/http"
	"net/url"
	"strings"
)

// newWebSocketCheckOrigin returns a CheckOrigin callback tailored to the runtime environment.
// - development: allow all origins to preserve local DX
// - production: allow only the origin derived from BASE_URL
func newWebSocketCheckOrigin(appEnv string, baseURL string) func(r *http.Request) bool {
	// Keep current local-dev behavior: accept all origins.
	if appEnv == "development" {
		return func(_ *http.Request) bool {
			return true
		}
	}

	allowedOrigin, ok := baseOriginFromURL(baseURL)
	// Security-first fallback in production:
	// if BASE_URL is invalid, reject websocket upgrades instead of allowing all origins.
	if !ok || allowedOrigin == "" {
		return func(_ *http.Request) bool {
			// If the origin is not allowed, return false.
			// This is to prevent any potential security issues.
			return false
		}
	}

	return func(r *http.Request) bool {
		// Browsers send Origin on websocket handshakes; empty origin is not accepted in production.
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			return false
		}

		// RFC 6454 origin comparison is ASCII case-insensitive.
		return strings.EqualFold(origin, allowedOrigin)
	}
}

func baseOriginFromURL(baseURL string) (string, bool) {
	// baseOriginFromURL extracts the canonical origin (scheme://host) from BASE_URL.
	// Example: https://app.example.com/path -> https://app.example.com
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", false
	}

	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", false
	}

	return parsedURL.Scheme + "://" + parsedURL.Host, true
}
