package hetzner

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const PresignedURLPayloadType = "hetzner.object.presigned"

const defaultExpiresInSeconds = 3600

type PresignedURL struct{}

type PresignedURLSpec struct {
	Bucket    string `json:"bucket" mapstructure:"bucket"`
	Key       any    `json:"key" mapstructure:"key"`
	ExpiresIn int    `json:"expiresIn" mapstructure:"expiresIn"`
	Method    string `json:"method" mapstructure:"method"`
}

func (c *PresignedURL) Name() string {
	return "hetzner.presignedUrl"
}

func (c *PresignedURL) Label() string {
	return "Presigned URL"
}

func (c *PresignedURL) Description() string {
	return "Generate a presigned URL for a Hetzner Object Storage object"
}

func (c *PresignedURL) Documentation() string {
	return `Generates a time-limited presigned URL for a Hetzner Object Storage object using the S3-compatible API.

## Requirements

S3 credentials (Access Key ID, Secret Access Key, and Region) must be configured on the Hetzner integration.

## Configuration

- **Bucket**: The bucket containing the object (dropdown or expression).
- **Key**: The object key (supports expressions).
- **Expires In** (optional): URL expiry in seconds. Defaults to 3600 (1 hour).
- **Method**: The HTTP method the presigned URL allows — GET (download) or PUT (upload).

## Output

Emits the bucket, key, the presigned URL, and the expiry timestamp (ISO 8601) on the default channel.

## Use cases

- Share a generated report or artifact with an external system via Slack/email without exposing credentials.
- Allow an external agent to upload a file to a specific bucket/key without permanent access.
`
}

func (c *PresignedURL) Icon() string {
	return "hetzner"
}

func (c *PresignedURL) Color() string {
	return "gray"
}

func (c *PresignedURL) ExampleOutput() map[string]any {
	return exampleOutputPresignedURL()
}

func (c *PresignedURL) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PresignedURL) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "bucket",
			Label:       "Bucket",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The bucket containing the object",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "bucket",
				},
			},
		},
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Object key",
		},
		{
			Name:     "method",
			Label:    "Method",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "GET",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GET (download)", Value: "GET"},
						{Label: "PUT (upload)", Value: "PUT"},
					},
				},
			},
			Description: "HTTP method the presigned URL will allow",
		},
		{
			Name:        "expiresIn",
			Label:       "Expires In (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultExpiresInSeconds,
			Description: "URL expiry in seconds (default: 3600 = 1 hour)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(60),
					Max: intPtr(604800), // 7 days
				},
			},
		},
	}
}

func (c *PresignedURL) Setup(ctx core.SetupContext) error {
	spec := PresignedURLSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.Bucket) == "" {
		return fmt.Errorf("bucket is required")
	}
	if strings.TrimSpace(readStringFromAny(spec.Key)) == "" {
		return fmt.Errorf("key is required")
	}
	return nil
}

func (c *PresignedURL) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PresignedURL) Execute(ctx core.ExecutionContext) error {
	spec := PresignedURLSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}
	bucket := strings.TrimSpace(spec.Bucket)
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	key := strings.TrimSpace(readStringFromAny(spec.Key))
	if key == "" {
		return fmt.Errorf("key is required")
	}
	method := strings.ToUpper(strings.TrimSpace(spec.Method))
	if method == "" {
		method = "GET"
	}
	expiresIn := spec.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = defaultExpiresInSeconds
	}

	s3, err := NewHetznerS3Client(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	signedURL, err := s3.PresignURL(bucket, key, method, time.Duration(expiresIn)*time.Second)
	if err != nil {
		return fmt.Errorf("generate presigned URL: %w", err)
	}

	expiresAt := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second).Format(time.RFC3339)
	payload := map[string]any{
		"bucket":    bucket,
		"key":       key,
		"url":       signedURL,
		"expiresAt": expiresAt,
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PresignedURLPayloadType, []any{payload})
}

func (c *PresignedURL) Actions() []core.Action                  { return nil }
func (c *PresignedURL) HandleAction(_ core.ActionContext) error { return nil }
func (c *PresignedURL) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (c *PresignedURL) Cancel(_ core.ExecutionContext) error { return nil }
func (c *PresignedURL) Cleanup(_ core.SetupContext) error    { return nil }
