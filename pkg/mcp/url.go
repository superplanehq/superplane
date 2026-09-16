package mcp

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

const maxRedirects = 3

// ValidatePublicHTTPSURL rejects empty, non-https, and private MCP endpoints.
func ValidatePublicHTTPSURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fmt.Errorf("MCP URL is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("MCP URL is not valid")
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("MCP URL must use https")
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("MCP URL must have a host")
	}
	if err := rejectPrivateHost(host); err != nil {
		return err
	}
	return nil
}

func rejectPrivateHost(host string) error {
	lower := strings.ToLower(strings.TrimSpace(host))
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return fmt.Errorf("MCP URL must not use a private host")
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("MCP URL must not use a private IP")
	}
	return nil
}

func parseHTTPSURL(raw string) (*url.URL, error) {
	if err := ValidatePublicHTTPSURL(raw); err != nil {
		return nil, err
	}
	return url.Parse(strings.TrimSpace(raw))
}
