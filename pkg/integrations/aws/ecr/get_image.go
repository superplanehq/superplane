package ecr

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type GetImage struct{}

type GetImageConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	Repository  string `json:"repository" mapstructure:"repository"`
	ImageDigest string `json:"imageDigest" mapstructure:"imageDigest"`
	ImageTag    string `json:"imageTag" mapstructure:"imageTag"`
}

func (c *GetImage) Name() string {
	return "aws.ecr.getImage"
}

func (c *GetImage) Label() string {
	return "ECR • Get Image"
}

func (c *GetImage) Description() string {
	return "Get an ECR image by digest or tag"
}

func (c *GetImage) Documentation() string {
	return `The Get Image component retrieves image metadata from an ECR repository by digest, tag, or both.

## Use Cases

- **Release automation**: Fetch image details before deployment
- **Audit trails**: Resolve digests and tags for traceability
- **Security workflows**: Enrich findings with image metadata

## Configuration

- **Region**: AWS region of the ECR repository
- **Repository**: ECR repository name or ARN
- **Image Digest**: Digest of the image (optional)
- **Image Tag**: Tag of the image (optional)

At least one of **Image Digest** or **Image Tag** is required. If both are provided, the request includes both.`
}

func (c *GetImage) Icon() string {
	return "aws"
}

func (c *GetImage) Color() string {
	return "gray"
}

func (c *GetImage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetImage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "us-east-1",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: common.AllRegions,
				},
			},
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "ECR repository name or ARN",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "ecr.repository",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "region",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "region",
							},
						},
					},
				},
			},
		},
		{
			Name:        "imageDigest",
			Label:       "Image Digest",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "sha256:...",
		},
		{
			Name:        "imageTag",
			Label:       "Image Tag",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "latest",
		},
	}
}

func (c *GetImage) Setup(ctx core.SetupContext) error {
	var config GetImageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	region := strings.TrimSpace(config.Region)
	if region == "" {
		return fmt.Errorf("region is required")
	}

	if strings.TrimSpace(config.Repository) == "" {
		return fmt.Errorf("repository is required")
	}

	if strings.TrimSpace(config.ImageDigest) == "" && strings.TrimSpace(config.ImageTag) == "" {
		return fmt.Errorf("image digest or image tag is required")
	}

	return nil
}

func (c *GetImage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetImage) Execute(ctx core.ExecutionContext) error {
	var config GetImageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	imageDetail, err := client.DescribeImage(config.Repository, config.ImageDigest, config.ImageTag)
	if err != nil {
		return fmt.Errorf("failed to describe image: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ecr.image",
		[]any{imageDetail},
	)
}

func (c *GetImage) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetImage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetImage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetImage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetImage) Cleanup(ctx core.SetupContext) error {
	return nil
}
