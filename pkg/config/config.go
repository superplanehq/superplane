package config

import (
	"fmt"
	"os"
)

func RepoProxyURL() (string, error) {
	URL := os.Getenv("INTERNAL_API_URL_REPO_PROXY")
	if URL == "" {
		return "", fmt.Errorf("INTERNAL_API_URL_REPO_PROXY not set")
	}

	return URL, nil
}

func RabbitMQURL() (string, error) {
	URL := os.Getenv("RABBITMQ_URL")
	if URL == "" {
		return "", fmt.Errorf("RABBITMQ_URL not set")
	}

	return URL, nil
}

func PipelineAPIURL() (string, error) {
	URL := os.Getenv("INTERNAL_API_URL_PLUMBER")
	if URL == "" {
		return "", fmt.Errorf("INTERNAL_API_URL_PLUMBER not set")
	}

	return URL, nil
}

func SchedulerAPIURL() (string, error) {
	URL := os.Getenv("INTERNAL_API_URL_SCHEDULER")
	if URL == "" {
		return "", fmt.Errorf("INTERNAL_API_URL_SCHEDULER not set")
	}

	return URL, nil
}
