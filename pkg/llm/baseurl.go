package llm

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func ValidateBaseURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid LLM base URL")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("LLM base URL must be http or https")
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("LLM base URL must have a host")
	}
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("LLM base URL must not use a private host")
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("LLM base URL must not use a private IP")
		}
	}

	return nil
}
