package sns

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

// requireRegion validates and normalizes region values.
func requireRegion(region string) (string, error) {
	normalized := strings.TrimSpace(region)
	if normalized == "" {
		return "", fmt.Errorf("region is required")
	}

	// Validate against known AWS regions
	for _, r := range common.AllRegions {
		if r.Value == normalized {
			return normalized, nil
		}
	}

	return "", fmt.Errorf("invalid AWS region: %s", normalized)
}

// requireTopicArn validates and normalizes topic ARNs.
func requireTopicArn(topicArn string) (string, error) {
	normalized := strings.TrimSpace(topicArn)
	if normalized == "" {
		return "", fmt.Errorf("topic ARN is required")
	}

	// Validate ARN format: arn:<partition>:sns:region:account-id:topic-name
	if !strings.HasPrefix(normalized, "arn:") {
		return "", fmt.Errorf("invalid topic ARN format: must start with 'arn:'")
	}

	parts := strings.Split(normalized, ":")
	if len(parts) < 6 {
		return "", fmt.Errorf("invalid topic ARN format: expected arn:<partition>:sns:region:account-id:topic-name")
	}

	if parts[2] != "sns" {
		return "", fmt.Errorf("invalid topic ARN format: expected SNS service ARN")
	}

	return normalized, nil
}

// requireSubscriptionArn validates and normalizes subscription ARNs.
func requireSubscriptionArn(subscriptionArn string) (string, error) {
	normalized := strings.TrimSpace(subscriptionArn)
	if normalized == "" {
		return "", fmt.Errorf("subscription ARN is required")
	}

	// Validate ARN format: arn:<partition>:sns:region:account-id:topic-name:subscription-id
	if !strings.HasPrefix(normalized, "arn:") {
		return "", fmt.Errorf("invalid subscription ARN format: must start with 'arn:'")
	}

	parts := strings.Split(normalized, ":")
	if len(parts) < 7 {
		return "", fmt.Errorf("invalid subscription ARN format: expected arn:<partition>:sns:region:account-id:topic-name:subscription-id")
	}

	if parts[2] != "sns" {
		return "", fmt.Errorf("invalid subscription ARN format: expected SNS service ARN")
	}

	return normalized, nil
}
