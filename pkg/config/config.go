package config

import (
	"fmt"
	"os"
)

const MaxWebhookPayloadSize = 512 * 1024

func RabbitMQURL() (string, error) {
	URL := os.Getenv("RABBITMQ_URL")
	if URL == "" {
		return "", fmt.Errorf("RABBITMQ_URL not set")
	}

	return URL, nil
}

func MaxEmitCount() int {
	return intFromEnv("SUPERPLANE_MAX_EMIT_COUNT", 100)
}

func MaxPayloadSize() int {
	return intFromEnv("SUPERPLANE_MAX_PAYLOAD_SIZE", 512*1024)
}
