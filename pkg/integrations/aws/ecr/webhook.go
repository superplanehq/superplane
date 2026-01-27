package ecr

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

const (
	WebhookEventImagePush = "ecr.image.push"
	apiKeyHeaderName      = "x-superplane-webhook-secret"
)

type WebhookConfiguration struct {
	EventType string `json:"eventType" mapstructure:"eventType"`
}

type WebhookMetadata struct {
	ConnectionArn        string `json:"connectionArn"`
	ConnectionName       string `json:"connectionName"`
	ApiDestinationArn    string `json:"apiDestinationArn"`
	ApiDestinationName   string `json:"apiDestinationName"`
	RuleArn              string `json:"ruleArn"`
	RuleName             string `json:"ruleName"`
	TargetID             string `json:"targetId"`
	InvocationURL        string `json:"invocationUrl"`
	InvocationAuthHeader string `json:"invocationAuthHeader"`
}

type webhookResourceNames struct {
	ConnectionName     string
	ApiDestinationName string
	RuleName           string
	TargetID           string
}

func CompareWebhookConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.EventType == configB.EventType, nil
}

func SetupWebhook(ctx core.SetupWebhookContext, config WebhookConfiguration) (any, error) {
	if strings.TrimSpace(config.EventType) == "" {
		return nil, fmt.Errorf("event type is required")
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(common.RegionFromInstallation(ctx.Integration))
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewEventBridgeClient(ctx.HTTP, creds, region)

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %w", err)
	}

	names := resourceNamesFromWebhookID(ctx.Webhook.GetID())

	connectionArn, err := ensureConnection(client, names.ConnectionName, secret)
	if err != nil {
		return nil, err
	}

	apiDestinationArn, err := ensureApiDestination(client, names.ApiDestinationName, connectionArn, ctx.Webhook.GetURL())
	if err != nil {
		return nil, err
	}

	eventPattern, err := eventPatternForType(config.EventType)
	if err != nil {
		return nil, err
	}

	patternBytes, err := json.Marshal(eventPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event pattern: %w", err)
	}

	ruleArn, err := client.PutRule(names.RuleName, string(patternBytes), "SuperPlane ECR webhook rule")
	if err != nil {
		return nil, err
	}

	if err := client.PutTargets(names.RuleName, []Target{
		{
			ID:  names.TargetID,
			Arn: apiDestinationArn,
		},
	}); err != nil {
		return nil, err
	}

	return WebhookMetadata{
		ConnectionArn:        connectionArn,
		ConnectionName:       names.ConnectionName,
		ApiDestinationArn:    apiDestinationArn,
		ApiDestinationName:   names.ApiDestinationName,
		RuleArn:              ruleArn,
		RuleName:             names.RuleName,
		TargetID:             names.TargetID,
		InvocationURL:        ctx.Webhook.GetURL(),
		InvocationAuthHeader: apiKeyHeaderName,
	}, nil
}

func CleanupWebhook(ctx core.CleanupWebhookContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("error decoding webhook metadata: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return err
	}

	region := strings.TrimSpace(common.RegionFromInstallation(ctx.Integration))
	if region == "" {
		return fmt.Errorf("region is required")
	}

	client := NewEventBridgeClient(ctx.HTTP, creds, region)

	if metadata.RuleName != "" && metadata.TargetID != "" {
		if err := client.RemoveTargets(metadata.RuleName, []string{metadata.TargetID}); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	if metadata.RuleName != "" {
		if err := client.DeleteRule(metadata.RuleName); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	if metadata.ApiDestinationName != "" {
		if err := client.DeleteApiDestination(metadata.ApiDestinationName); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	if metadata.ConnectionName != "" {
		if err := client.DeleteConnection(metadata.ConnectionName); err != nil && !isNotFoundError(err) {
			return err
		}
	}

	return nil
}

func ensureConnection(client *EventBridgeClient, name string, secret []byte) (string, error) {
	connectionArn, err := client.CreateConnection(name, apiKeyHeaderName, string(secret))
	if err == nil {
		return connectionArn, nil
	}

	if !isAlreadyExistsError(err) {
		return "", err
	}

	return client.DescribeConnection(name)
}

func ensureApiDestination(client *EventBridgeClient, name, connectionArn, url string) (string, error) {
	apiDestinationArn, err := client.CreateApiDestination(name, connectionArn, url)
	if err == nil {
		return apiDestinationArn, nil
	}

	if !isAlreadyExistsError(err) {
		return "", err
	}

	return client.DescribeApiDestination(name)
}

func resourceNamesFromWebhookID(webhookID string) webhookResourceNames {
	suffix := strings.ReplaceAll(webhookID, "-", "")
	if len(suffix) > 12 {
		suffix = suffix[:12]
	}

	base := fmt.Sprintf("superplane-ecr-%s", suffix)
	return webhookResourceNames{
		ConnectionName:     truncateName(fmt.Sprintf("%s-conn", base)),
		ApiDestinationName: truncateName(fmt.Sprintf("%s-dest", base)),
		RuleName:           truncateName(fmt.Sprintf("%s-rule", base)),
		TargetID:           truncateName(fmt.Sprintf("%s-target", base)),
	}
}

func truncateName(name string) string {
	const maxLength = 64
	if len(name) <= maxLength {
		return name
	}

	return name[:maxLength]
}

func eventPatternForType(eventType string) (map[string]any, error) {
	switch eventType {
	case WebhookEventImagePush:
		return map[string]any{
			"source":      []string{"aws.ecr"},
			"detail-type": []string{"ECR Image Action"},
			"detail": map[string]any{
				"action-type": []string{"PUSH"},
				"result":      []string{"SUCCESS"},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}
}

func isAlreadyExistsError(err error) bool {
	var awsErr *awsError
	if errors.As(err, &awsErr) {
		return strings.Contains(awsErr.Code, "ResourceAlreadyExists")
	}

	return false
}

func isNotFoundError(err error) bool {
	var awsErr *awsError
	if errors.As(err, &awsErr) {
		return strings.Contains(awsErr.Code, "ResourceNotFound")
	}

	return false
}
