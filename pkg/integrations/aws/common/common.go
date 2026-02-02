package common

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	accessKeyIDSecret     = "accessKeyId"
	secretAccessKeySecret = "secretAccessKey"
	sessionTokenSecret    = "sessionToken"
)

type IntegrationMetadata struct {
	Session     *SessionMetadata     `json:"session" mapstructure:"session"`
	IAM         *IAMMetadata         `json:"iam" mapstructure:"iam"`
	EventBridge *EventBridgeMetadata `json:"eventBridge" mapstructure:"eventBridge"`
	Tags        []Tag                `json:"tags" mapstructure:"tags"`
}

type SessionMetadata struct {
	RoleArn   string `json:"roleArn"`
	AccountID string `json:"accountId"`
	Region    string `json:"region"`
	ExpiresAt string `json:"expiresAt"`
}

/*
 * IAM metadata for the integration.
 */
type IAMMetadata struct {

	/*
	 * The role ARN of the role that will be used to invoke the EventBridge API destinations.
	 */
	TargetDestinationRoleArn string `json:"targetDestinationRoleArn" mapstructure:"targetDestinationRoleArn"`
}

/*
 * EventBridge metadata for the integration.
 */
type EventBridgeMetadata struct {

	/*
	 * Since we need to support multiple regions,
	 * the integration needs to maintain one connection/destination per region.
	 */
	APIDestinations map[string]APIDestinationMetadata `json:"apiDestinations" mapstructure:"apiDestinations"`
}

type APIDestinationMetadata struct {
	ConnectionArn     string `json:"connectionArn"`
	ApiDestinationArn string `json:"apiDestinationArn"`
}

type ProvisionDestinationParameters struct {
	Region string `json:"region"`
}

type EventBridgeEvent struct {
	Region     string         `json:"region"`
	DetailType string         `json:"detail-type"`
	Source     string         `json:"source"`
	Detail     map[string]any `json:"detail"`
}

type Tag struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

func TagsForAPI(tags []Tag) []any {
	apiTags := make([]any, len(tags))
	for i, tag := range tags {
		apiTags[i] = map[string]string{
			"Key":   tag.Key,
			"Value": tag.Value,
		}
	}
	return apiTags
}

func CredentialsFromInstallation(ctx core.IntegrationContext) (aws.Credentials, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to get AWS session secrets: %w", err)
	}

	var accessKeyID string
	var secretAccessKey string
	var sessionToken string

	for _, secret := range secrets {
		switch secret.Name {
		case accessKeyIDSecret:
			accessKeyID = string(secret.Value)
		case secretAccessKeySecret:
			secretAccessKey = string(secret.Value)
		case sessionTokenSecret:
			sessionToken = string(secret.Value)
		}
	}

	if strings.TrimSpace(accessKeyID) == "" || strings.TrimSpace(secretAccessKey) == "" || strings.TrimSpace(sessionToken) == "" {
		return aws.Credentials{}, fmt.Errorf("AWS session credentials are missing")
	}

	return aws.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
		Source:          "superplane",
	}, nil
}

func RegionFromInstallation(ctx core.IntegrationContext) string {
	regionBytes, err := ctx.GetConfig("region")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(regionBytes))
}

func NormalizeTags(tags []Tag) []Tag {
	if len(tags) == 0 {
		return nil
	}

	normalized := make([]Tag, 0, len(tags))
	seen := map[string]int{}
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}

		value := strings.TrimSpace(tag.Value)
		if index, ok := seen[key]; ok {
			normalized[index].Value = value
			continue
		}

		seen[key] = len(normalized)
		normalized = append(normalized, Tag{
			Key:   key,
			Value: value,
		})
	}

	return normalized
}

/*
 * Extract the account ID from an IAM role ARN.
 *
 * Expected format: arn:aws:iam::<account-id>:role/<role-name>
 */
func AccountIDFromRoleArn(roleArn string) (string, error) {
	roleArn = strings.TrimSpace(roleArn)
	if roleArn == "" {
		return "", fmt.Errorf("role ARN is empty")
	}

	parts := strings.Split(roleArn, ":")
	if len(parts) < 6 {
		return "", fmt.Errorf("role ARN is invalid")
	}

	if parts[0] != "arn" {
		return "", fmt.Errorf("role ARN is invalid")
	}

	return strings.TrimSpace(parts[4]), nil
}
