package ec2

import (
	"fmt"
	"strings"
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
