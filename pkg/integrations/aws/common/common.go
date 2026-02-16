package common

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	accessKeyIDSecret     = "accessKeyId"
	secretAccessKeySecret = "secretAccessKey"
	sessionTokenSecret    = "sessionToken"
)

var AllRegions = []configuration.FieldOption{
	{
		Label: "us-east-1",
		Value: "us-east-1",
	},
	{
		Label: "us-east-2",
		Value: "us-east-2",
	},
	{
		Label: "us-west-1",
		Value: "us-west-1",
	},
	{
		Label: "us-west-2",
		Value: "us-west-2",
	},
	{
		Label: "eu-west-1",
		Value: "eu-west-1",
	},
	{
		Label: "eu-central-1",
		Value: "eu-central-1",
	},
	{
		Label: "ap-northeast-1",
		Value: "ap-northeast-1",
	},
	{
		Label: "ap-northeast-2",
		Value: "ap-northeast-2",
	},
	{
		Label: "ap-southeast-1",
		Value: "ap-southeast-1",
	},
	{
		Label: "ap-southeast-2",
		Value: "ap-southeast-2",
	},
	{
		Label: "ap-south-1",
		Value: "ap-south-1",
	},
	{
		Label: "ca-central-1",
		Value: "ca-central-1",
	},
	{
		Label: "cn-north-1",
		Value: "cn-north-1",
	},
	{
		Label: "cn-northwest-1",
		Value: "cn-northwest-1",
	},
	{
		Label: "eu-north-1",
		Value: "eu-north-1",
	},
	{
		Label: "eu-south-1",
		Value: "eu-south-1",
	},
	{
		Label: "eu-west-2",
		Value: "eu-west-2",
	},
	{
		Label: "eu-west-3",
		Value: "eu-west-3",
	},
	{
		Label: "sa-east-1",
		Value: "sa-east-1",
	},
}

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
	TargetDestinationRole *IAMRoleMetadata `json:"targetDestinationRole" mapstructure:"targetDestinationRole"`
}

type IAMRoleMetadata struct {
	RoleArn  string `json:"roleArn" mapstructure:"roleArn"`
	RoleName string `json:"roleName" mapstructure:"roleName"`
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

	/*
	 * List of EventBridge rules created by the integration.
	 * This ensures that we reuse the same rule for the same source, e.g., aws.codeartifact, aws.ecr, etc.
	 */
	Rules map[string]EventBridgeRuleMetadata `json:"rules" mapstructure:"rules"`
}

type EventBridgeRuleMetadata struct {
	Source      string   `json:"source" mapstructure:"source"`
	Region      string   `json:"region" mapstructure:"region"`
	Name        string   `json:"name" mapstructure:"name"`
	RuleArn     string   `json:"ruleArn" mapstructure:"ruleArn"`
	DetailTypes []string `json:"detailTypes" mapstructure:"detailTypes"`
}

type APIDestinationMetadata struct {
	Name              string `json:"name" mapstructure:"name"`
	Region            string `json:"region" mapstructure:"region"`
	ConnectionArn     string `json:"connectionArn" mapstructure:"connectionArn"`
	APIDestinationArn string `json:"apiDestinationArn" mapstructure:"apiDestinationArn"`
}

type ProvisionRuleParameters struct {
	Region     string `json:"region"`
	Source     string `json:"source"`
	DetailType string `json:"detailType"`
}

type EventBridgeEvent struct {
	Region     string         `json:"region" mapstructure:"region"`
	DetailType string         `json:"detail-type" mapstructure:"detail-type"`
	Source     string         `json:"source" mapstructure:"source"`
	Detail     map[string]any `json:"detail" mapstructure:"detail"`
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

func CredentialsFromInstallation(ctx core.IntegrationContext) (*aws.Credentials, error) {
	if ctx == nil {
		return nil, fmt.Errorf("AWS integration context is missing")
	}
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS session secrets: %w", err)
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
		return nil, fmt.Errorf("AWS session credentials are missing")
	}

	return &aws.Credentials{
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
