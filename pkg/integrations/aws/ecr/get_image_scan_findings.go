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

type GetImageScanFindings struct{}

type GetImageScanFindingsConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	Repository  string `json:"repository" mapstructure:"repository"`
	ImageDigest string `json:"imageDigest" mapstructure:"imageDigest"`
	ImageTag    string `json:"imageTag" mapstructure:"imageTag"`
}

func (c *GetImageScanFindings) Name() string {
	return "aws.ecr.getImageScanFindings"
}

func (c *GetImageScanFindings) Label() string {
	return "ECR • Get Image Scan Findings"
}

func (c *GetImageScanFindings) Description() string {
	return "Get ECR image scan findings by digest or tag"
}

func (c *GetImageScanFindings) Documentation() string {
	return `The Get Image Scan Findings component retrieves vulnerability scan results for an ECR image.

## Use Cases

- **Security automation**: Pull scan findings to drive alerting or approvals
- **Compliance checks**: Validate images against severity thresholds
- **Reporting**: Capture scan summaries and findings for audits

## Configuration

- **Region**: AWS region of the ECR repository
- **Repository**: ECR repository name or ARN
- **Image Digest**: Digest of the image (optional)
- **Image Tag**: Tag of the image (optional)

At least one of **Image Digest** or **Image Tag** is required. If both are provided, the request includes both.`
}

func (c *GetImageScanFindings) Icon() string {
	return "aws"
}

func (c *GetImageScanFindings) Color() string {
	return "gray"
}

func (c *GetImageScanFindings) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetImageScanFindings) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "region",
			Label:    "Region",
			Type:     configuration.FieldTypeString,
			Required: true,
			Default:  "us-east-1",
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "ECR repository name or ARN",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "ecr.repository",
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

func (c *GetImageScanFindings) Setup(ctx core.SetupContext) error {
	var config GetImageScanFindingsConfiguration
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

func (c *GetImageScanFindings) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetImageScanFindings) Execute(ctx core.ExecutionContext) error {
	var config GetImageScanFindingsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, creds, config.Region)
	response, err := client.DescribeImageScanFindings(config.Repository, config.ImageDigest, config.ImageTag)
	if err != nil {
		return fmt.Errorf("failed to describe image scan findings: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.ecr.image.scanFindings",
		[]any{response},
	)
}

func (c *GetImageScanFindings) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetImageScanFindings) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetImageScanFindings) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetImageScanFindings) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetImageScanFindings) Cleanup(ctx core.SetupContext) error {
	return nil
}
