package runner

import (
	"net"
	"net/url"
	"strings"
)

const localComposeFleetID = "local"

func isLocalTaskBrokerURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "host.docker.internal" || host == "localhost" || host == "127.0.0.1"
}

func browserTaskBrokerBaseURL(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if !strings.EqualFold(parsed.Hostname(), "host.docker.internal") {
		return raw
	}
	port := parsed.Port()
	if port == "" {
		parsed.Host = "localhost"
	} else {
		parsed.Host = net.JoinHostPort("localhost", port)
	}
	return strings.TrimRight(parsed.String(), "/")
}
