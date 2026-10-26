package runner

import (
	"net"
	"net/url"
	"os"
	"strings"
)

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
