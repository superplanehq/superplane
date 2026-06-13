package runner

import (
	"strings"
)

func taskLogEndpoint(baseURL string) (string, string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", "", nil
	}

	secret, err := taskBrokerAuthToken()
	if err != nil {
		return "", "", err
	}

	return baseURL + "/api/v1/runner-logs", secret, nil
}
