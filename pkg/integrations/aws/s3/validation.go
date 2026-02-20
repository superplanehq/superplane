package s3

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func requireRegion(region string) (string, error) {
	normalized := strings.TrimSpace(region)
	if normalized == "" {
		return "", fmt.Errorf("region is required")
	}

	for _, r := range common.AllRegions {
		if r.Value == normalized {
			return normalized, nil
		}
	}

	return "", fmt.Errorf("invalid AWS region: %s", normalized)
}

func requireBucket(bucket string) (string, error) {
	normalized := strings.TrimSpace(bucket)
	if normalized == "" {
		return "", fmt.Errorf("bucket name is required")
	}

	return normalized, nil
}

func requireKey(key string) (string, error) {
	normalized := strings.TrimSpace(key)
	if normalized == "" {
		return "", fmt.Errorf("object key is required")
	}

	return normalized, nil
}
