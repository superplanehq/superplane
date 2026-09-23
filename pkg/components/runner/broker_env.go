package runner

import (
	"net"
	"net/url"
	"os"
	"strings"
)

const localComposeFleetID = "local"

func isLocalTaskBrokerURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "host.docker.internal" || host == "localhost" || host == "127.0.0.1" || host == "task-broker"
}

func browserTaskBrokerBaseURL(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "task-broker" {
		parsed.Host = net.JoinHostPort("localhost", localTaskBrokerHostPort())
		return strings.TrimRight(parsed.String(), "/")
	}
	if host != "host.docker.internal" {
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

func localTaskBrokerHostPort() string {
	port := strings.TrimSpace(os.Getenv("TASK_BROKER_HOST_PORT"))
	if port == "" {
		return "8091"
	}
	return port
}
