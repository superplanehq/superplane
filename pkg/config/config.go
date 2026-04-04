package config

import (
	"fmt"
	"os"
	"strings"
)

func RabbitMQURL() (string, error) {
	URL := os.Getenv("RABBITMQ_URL")
	if URL == "" {
		return "", fmt.Errorf("RABBITMQ_URL not set")
	}

	return URL, nil
}

func UsageGRPCURL() string {
	return os.Getenv("USAGE_GRPC_URL")
}

func AgentHTTPURL() string {
	return os.Getenv("AGENT_HTTP_URL")
}

func ConfigAssistantHTTPURL() string {
	return strings.TrimSpace(os.Getenv("CONFIG_ASSISTANT_HTTP_URL"))
}

func AgentGRPCURL() string {
	return os.Getenv("AGENT_GRPC_URL")
}
