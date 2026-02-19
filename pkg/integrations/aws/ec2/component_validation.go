package ec2

import (
	"fmt"
	"strings"
	"time"
)

func requireRegion(value string) (string, error) {
	region := strings.TrimSpace(value)
	if region == "" {
		return "", fmt.Errorf("region is required")
	}
	return region, nil
}

func requireImageID(value string) (string, error) {
	imageID := strings.TrimSpace(value)
	if imageID == "" {
		return "", fmt.Errorf("image ID is required")
	}
	return imageID, nil
}

func requireSourceRegion(value string) (string, error) {
	region := strings.TrimSpace(value)
	if region == "" {
		return "", fmt.Errorf("source region is required")
	}
	return region, nil
}

func requireSourceImageID(value string) (string, error) {
	imageID := strings.TrimSpace(value)
	if imageID == "" {
		return "", fmt.Errorf("source image ID is required")
	}
	return imageID, nil
}

func requireImageName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", fmt.Errorf("image name is required")
	}
	return name, nil
}

func normalizeOptionalString(value string) string {
	return strings.TrimSpace(value)
}

func requireDeprecateAt(value string) (string, error) {
	deprecateAt := strings.TrimSpace(value)
	if deprecateAt == "" {
		return "", fmt.Errorf("deprecateAt is required")
	}

	parsed, err := time.Parse(time.RFC3339, deprecateAt)
	if err != nil {
		return "", fmt.Errorf("deprecateAt must be a valid RFC3339 timestamp")
	}

	return parsed.UTC().Format(time.RFC3339), nil
}
