package sns

import (
	"fmt"
	"slices"
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

	// Validate ARN format: arn:aws:sns:region:account-id:topic-name
	if !strings.HasPrefix(normalized, "arn:aws:sns:") {
		return "", fmt.Errorf("invalid topic ARN format: must start with 'arn:aws:sns:'")
	}

	parts := strings.Split(normalized, ":")
	if len(parts) < 6 {
		return "", fmt.Errorf("invalid topic ARN format: expected arn:aws:sns:region:account-id:topic-name")
	}

	return normalized, nil
}

// requireSubscriptionArn validates and normalizes subscription ARNs.
func requireSubscriptionArn(subscriptionArn string) (string, error) {
	normalized := strings.TrimSpace(subscriptionArn)
	if normalized == "" {
		return "", fmt.Errorf("subscription ARN is required")
	}

	// Validate ARN format: arn:aws:sns:region:account-id:topic-name:subscription-id
	if !strings.HasPrefix(normalized, "arn:aws:sns:") {
		return "", fmt.Errorf("invalid subscription ARN format: must start with 'arn:aws:sns:'")
	}

	parts := strings.Split(normalized, ":")
	if len(parts) < 7 {
		return "", fmt.Errorf("invalid subscription ARN format: expected arn:aws:sns:region:account-id:topic-name:subscription-id")
	}

	return normalized, nil
}

// requireTopicName validates and normalizes topic names.
func requireTopicName(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", fmt.Errorf("topic name is required")
	}

	return normalized, nil
}

// requireMessage validates and normalizes SNS publish message values.
func requireMessage(message string) (string, error) {
	normalized := strings.TrimSpace(message)
	if normalized == "" {
		return "", fmt.Errorf("message is required")
	}

	return normalized, nil
}

// requireProtocol validates SNS subscription protocol values.
func requireProtocol(protocol string) (string, error) {
	normalized := strings.TrimSpace(protocol)
	if normalized == "" {
		return "", fmt.Errorf("protocol is required")
	}

	allowedProtocols := []string{
		"email",
		"email-json",
		"http",
		"https",
		"sqs",
		"sms",
		"lambda",
		"application",
		"firehose",
	}
	if !slices.Contains(allowedProtocols, normalized) {
		return "", fmt.Errorf("unsupported protocol: %s", normalized)
	}

	return normalized, nil
}

// requireEndpoint validates and normalizes SNS subscription endpoints.
func requireEndpoint(endpoint string) (string, error) {
	normalized := strings.TrimSpace(endpoint)
	if normalized == "" {
		return "", fmt.Errorf("endpoint is required")
	}

	return normalized, nil
}
